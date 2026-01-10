document.body.addEventListener('htmx:sendError', onError);
document.body.addEventListener('htmx:responseError', onError);
document.body.addEventListener('htmx:afterOnLoad', onSuccess);

function onError(evt) { evt.detail.target.classList.add("request-error"); }
function onSuccess(evt) { evt.detail.target.classList.remove("request-error"); }
