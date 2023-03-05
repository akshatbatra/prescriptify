import QrScanner from "https://nimiq.github.io/qr-scanner/qr-scanner.min.js";

function processScannedCode(result, vidElem, qrElem, qrScanner) {
    qrScanner.destroy();
    let data = {qr_data: btoa(result.data), prescription_id: document.querySelector('input[name="prescription-id"]').value};
    fetch("/prescription/link", {
        method: "POST",
        body: JSON.stringify(data)
    })
      .then((response) => {
          if (response.ok) {
              vidElem.style.display = "none";
              qrElem.style.display = "block";
              let linkButton = document.getElementById("link-aadhaar-button");
              linkButton.removeEventListener('click', startScanner);
              linkButton.innerText = "Linked";
              linkButton.style.backgroundColor = "blue";
          }
      });
}

function startScanner() {
    let vidElem = document.getElementById("scanner");
    vidElem.style.display = "block";
    let qrElem = document.getElementById("qr-code");
    qrElem.style.display = "none";
    const qrScanner = new QrScanner(
        vidElem,
        result => processScannedCode(result, vidElem, qrElem, qrScanner),
        {preferredCamera: 'environment'}
    );
    qrScanner.start();
}
document.getElementById("link-aadhaar-button").addEventListener('click', startScanner);