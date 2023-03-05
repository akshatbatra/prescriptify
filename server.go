package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var (
	HOST     = os.Getenv("prescriptify_db_host")
	PORT     = "5432"
	DATABASE = "prescriptify"
	USER     = os.Getenv("prescriptify_db_admin")
	PASSWORD = os.Getenv("prescriptify_db_password")
	SSLMODE  = "require"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", HOST, PORT, USER, PASSWORD, DATABASE, SSLMODE))
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", serveHomePage)
	// separate endpoints for static files for increased security (better control on individual file access)
	http.HandleFunc("/css/main.css", serveICss)
	http.HandleFunc("/js/index.js", serveIJs)
	http.HandleFunc("/css/create_prescription.css", serveCpCss)
	http.HandleFunc("/js/create_prescription.js", serveCpJs)
	http.HandleFunc("/create_prescription", serveCpPage)
	http.HandleFunc("/css/see_prescription.css", serveSpCss)
	http.HandleFunc("/js/see_prescription.js", serveSpJs)
	http.HandleFunc("/medicine", queryMeds)
	http.HandleFunc("/submit_prescription", submitPrescription)
	http.HandleFunc("/prescription/", displayPrescription)
	http.HandleFunc("/prescription/link", linkWithQR)
	http.HandleFunc("/prescription/get_linked", scanQR)
	http.ListenAndServe(":8080", nil)
}

func cacheFix(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Pragma", "no-cache")
}

func serveICss(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "css/main.css")
}

func serveIJs(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "js/index.js")
}

func serveHomePage(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "templates/index.html")
}

func serveCpCss(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "css/create_prescription.css")
}

func serveCpJs(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "js/create_prescription.js")
}

func serveCpPage(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "templates/create_prescription.html")
}

func serveSpCss(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "css/see_prescription.css")
}

func serveSpJs(w http.ResponseWriter, r *http.Request) {
	cacheFix(w)
	http.ServeFile(w, r, "js/see_prescription.js")
}

// query medicines from Tata 1MG online pharmacy API
func queryMeds(w http.ResponseWriter, r *http.Request) {
	search_term := r.URL.Query().Get("query")
	resp, err := http.Get("https://www.1mg.com/api/v1/search/autocomplete?city=New%2520Delhi&pageSize=20&types=all&name=" + search_term)
	if err != nil {
		log.Fatal(err)
	}
	dat, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}

type prescriptionData struct {
	ImgLink  string
	Name     string
	Quantity float64
	Price    float64
}

type prescriptionTemplateDat struct {
	PrescriptionDataList   []*prescriptionData
	PrescriptionId         string
	FromCreatePrescription bool
}

// submit prescription after selecting medicines along with the right quantity
func submitPrescription(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	prescriptionId := uuid.New().String()
	var prescriptionDataList []*prescriptionData
	for i := range r.Form["name"] {
		prescriptionEntry := []string{r.Form["name"][i], r.Form["quantity"][i], r.Form["product-id"][i]}
		var tmp map[string]interface{}
		resp, err := http.Get("https://www.1mg.com/api/v1/search/autocomplete?city=New%2520Delhi&pageSize=20&types=all&name=" + url.QueryEscape(prescriptionEntry[0]))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("https://www.1mg.com/api/v1/search/autocomplete?city=New%2520Delhi&pageSize=20&types=all&name=" + url.QueryEscape(prescriptionEntry[0]))
		bodyData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal(bodyData, &tmp)
		defer resp.Body.Close()

		for _, result := range tmp["results"].([]interface{}) {
			receivedProductId, _ := strconv.ParseFloat(prescriptionEntry[2], 0)
			if result.(map[string]interface{})["id"].(float64) == float64(receivedProductId) {
				var moq float64
				switch tmpVal := result.(map[string]interface{})["min_order_qty"].(type) {
				case nil:
					moq = 1
				default:
					moq = tmpVal.(float64)
				}
				pricePerUnit := result.(map[string]interface{})["price"].(float64) / (result.(map[string]interface{})["units_in_pack"].(float64) / moq)
				quantity, _ := strconv.ParseFloat(prescriptionEntry[1], 0)
				priceForCurrentOrder := math.Round(pricePerUnit*quantity*100) / 100
				imgLinks := result.(map[string]interface{})["cropped_image_urls"].([]interface{})
				var imgLink string
				if len(imgLinks) > 0 {
					imgLink = imgLinks[0].(string)
				} else {
					imgLink = "https://onemg.gumlet.io/a_ignore,w_380,h_380,c_fit,q_auto,f_auto/hx2gxivwmeoxxxsc1hix.png"
				}
				_, err = db.Query("INSERT INTO prescription_entries(prescription_id, product_id, img_link, name, quantity, price) VALUES ($1, $2, $3, $4, $5, $6)", prescriptionId, prescriptionEntry[2], imgLink, prescriptionEntry[0], quantity, priceForCurrentOrder)
				if err != nil {
					log.Fatal(err)
				}
				prescriptionDataList = append([]*prescriptionData{{ImgLink: imgLink, Name: prescriptionEntry[0], Quantity: quantity, Price: priceForCurrentOrder}}, prescriptionDataList...)
			}
		}

	}

	t, err := template.ParseFiles("templates/see_prescription.html")
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "text/html")
	templateData := &prescriptionTemplateDat{PrescriptionDataList: prescriptionDataList, PrescriptionId: prescriptionId, FromCreatePrescription: true}
	fmt.Printf("%v", t)
	err = t.Execute(w, templateData)
	if err != nil {
		log.Fatal(err)
	}
}

// render display page for prescription medicines (identified by a prescription ID)
func displayPrescription(w http.ResponseWriter, r *http.Request) {
	prescriptionIdSplit := strings.Split(r.URL.Path, "/")
	prescriptionId := prescriptionIdSplit[len(prescriptionIdSplit)-1]
	rows, err := db.Query("SELECT img_link, name, quantity, price FROM prescription_entries WHERE prescription_id = $1 ORDER BY created_at", prescriptionId)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var prescriptionList []*prescriptionData
	for rows.Next() {
		var pData prescriptionData
		if err := rows.Scan(&pData.ImgLink, &pData.Name, &pData.Quantity, &pData.Price); err != nil {
			log.Fatal(err)
		}
		prescriptionList = append(prescriptionList, &pData)
	}
	var fromCP bool
	if r.URL.Query().Get("from") == "create_prescription" {
		fromCP = true
	}
	templateData := &prescriptionTemplateDat{PrescriptionDataList: prescriptionList, PrescriptionId: prescriptionId, FromCreatePrescription: fromCP}
	cacheFix(w)
	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFiles("templates/see_prescription.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, templateData)
}

// link a prescription with a QR code (e.g. QR on Aadhaar card ~ Indian Identity card)
func linkWithQR(w http.ResponseWriter, r *http.Request) {
	var dat map[string]string
	reqBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(reqBody, &dat)
	checksum := sha512.Sum512([]byte(dat["qr_data"]))
	qr_data := fmt.Sprintf("%x", checksum)
	fmt.Println(qr_data)
	_, err = db.Query("INSERT INTO prescription_qr_mapping(qr_data, prescription_id) VALUES ($1, $2)", qr_data, dat["prescription_id"])
	if err != nil {
		log.Fatal(err)
	}
	retData := make(map[string]string)
	retData["qr_data"] = qr_data
	toReturn, err := json.Marshal(retData)
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(toReturn)
}

// scan a QR code for redirecting to the prescription display page
func scanQR(w http.ResponseWriter, r *http.Request) {
	var dat map[string]string
	reqBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(reqBody, &dat)
	retDat := make(map[string]string)
	var prescriptionId string
	checksum := sha512.Sum512([]byte(dat["qr_data"]))
	qr_data := fmt.Sprintf("%x", checksum)
	if err := db.QueryRow("SELECT prescription_id FROM prescription_qr_mapping WHERE qr_data = $1 ORDER BY created_at DESC", qr_data).Scan(&prescriptionId); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Fatal(err)
	}
	retDat["prescription_id"] = prescriptionId
	toReturn, err := json.Marshal(retDat)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(toReturn)
}
