function typeWriter (txt, index, elem) {
    elem.textContent += txt[index];
    setTimeout(function () {
        if (index !== txt.length -1) {
            typeWriter(txt, index+1, elem);
        }
    }, 95);
}
window.onload = (event) => {
    let txt = "Revolutionizing prescriptions for good by delivering transparency.";
    let elem = document.getElementById("slogan-text");
    typeWriter(txt, 0, elem);
}