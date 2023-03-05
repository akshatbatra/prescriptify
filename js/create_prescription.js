const speechRecognition = window.SpeechRecognition || webkitSpeechRecognition;
const speechRecognitionEvent = window.SpeechRecognitionEvent || webkitSpeechRecognitionEvent;
const recognition = new speechRecognition();
recognition.lang = 'en-US';
recognition.onresult = (event) => {
    const txt =  event.results[0][0].transcript;
    let elem = document.getElementById("med-search");
    elem.value = txt;
    let ev = new Event("input");
    elem.dispatchEvent(ev);
}
recognition.onspeechend = () => {
    let elem = document.querySelector(".fa-microphone-lines");
    if (elem !== null) {
        elem.classList.remove("fa-microphone-lines");
        elem.classList.add("fa-microphone-lines-slash");
    }
    recognition.stop();
}

function toggleListening(elem) {
    if (elem.classList.contains("fa-microphone-lines-slash")) {
        elem.classList.remove("fa-microphone-lines-slash");
        elem.classList.add("fa-microphone-lines");
        recognition.start();
    }
    else {
        elem.classList.remove("fa-microphone-lines");
        elem.classList.add("fa-microphone-lines-slash");
        recognition.stop();
    }
}

function updatePrice(elem) {
    let parentElem = elem.parentElement.parentElement;
    console.log("here", parseFloat($(".per_unit_price", $(parentElem)).val()));
    $(".price-text", $(parentElem)).text((parseFloat($(".per_unit_price", $(parentElem)).val()) * parseFloat(elem.value)).toFixed(2));
}

function addToPrescription(elem) {
    if ($('#prescription-table').length < 1) {
        document.getElementById("main-content").innerHTML += `<form id="prescription-form" action="submit_prescription" method="POST"><table id="prescription-table">
            <tr>
            <th>Item</th>
            <!-- <th>Name</th> -->
            <th>Quantity</th>
            <th>Price</th>
            </tr>
        </table></form><a style="text-align: center;"><button form="prescription-form" type="submit" id="generate-button">Generate Prescription</button></a>`
    }

    let label = $('.drop-elem-text', $(elem)).text();
    let product_id = $('.product-id', $(elem)).attr('value');
    let img_src = $('img', $(elem)).attr('src');
    let price = $('.price', $(elem)).attr('value');
    let units_in_pack = $('.ui-pack', $(elem)).attr('value');
    let min_order_quantity = $('.min-oq', $(elem)).attr('value');
    let one_pack_units = parseInt(units_in_pack) / parseInt(min_order_quantity);
    let per_unit_price = parseFloat(price) / one_pack_units;
    document.getElementById("med-dropdown").innerHTML = "";

    let table = document.getElementById("prescription-table");

    table.innerHTML += `
        <tr>
           <td class="item"><img src='${img_src}' class='img-max table-item-img'></img><br>${label}</td>
           <!-- <td class="name">${label}</td> -->
           <input type="hidden" name="name" value="${label}"></input>
           <td><input type="text" value="${one_pack_units}" style="width:30px;" oninput="updatePrice(this);" name="quantity" class="quantity"></input></td>
           <td class="price"><a href="#" onclick="this.parentElement.parentElement.remove();"><div class="cross">X</div></a><div class="price-text">${price}</div></td>
           <input class="per_unit_price" type="hidden" value="${per_unit_price}"></input>
           <input class="product-id" name="product-id" type="hidden" value="${product_id}"></input>
         </tr>     
    `;

    document.getElementById("med-search").value = "";
    document.getElementById("med-search").addEventListener("input" , suggestions);
}

function suggestions (event) {

    fetch("medicine?query=" + encodeURIComponent(event.target.value))
        .then((response) => response.json())
        .then((data) => {
             let results = data["results"];
             let elem = document.getElementById("med-dropdown");
             elem.innerHTML = "";

             let counter = 0;
             for (let i = 0; i < results.length; i++) {
                 if (results[i].hasOwnProperty('cropped_image_urls')) {

                     if (results[i]["min_order_qty"] === null) {
                         results[i]["min_order_qty"] = 1;
                     }

                     elem.innerHTML += `
                        <div class="dropdown-element" onclick='addToPrescription(this);'>
                            <img src="${results[i]['cropped_image_urls'][0]}" class="img-max"></img>
                            <span class="drop-elem-text">${results[i]['label']}</span>
                            <input type='hidden' class='price' value='${results[i]['price']}'></input>
                            <input type='hidden' class='ui-pack' value='${results[i]['units_in_pack']}'></input>
                            <input type='hidden' class='min-oq' value='${results[i]['min_order_qty']}'></input>
                            <input type='hidden' class='product-id' value='${results[i]['id']}'></input>
                        </div>`;

                     counter += 1;
                     if (counter === 3) {
                         break;
                     }
                 }
             }

         });
    document.getElementById("med-search").addEventListener("input" , suggestions);
}