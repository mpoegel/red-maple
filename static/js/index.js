document.body.addEventListener('htmx:sendError', function (evt) {
    evt.detail.target.classList.add("request-error");
});

document.body.addEventListener('htmx:afterOnLoad', function (evt) {
    evt.detail.target.classList.remove("request-error");
});
