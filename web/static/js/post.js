var postForm = document.forms.namedItem("post");
postForm.addEventListener("submit", function() {
	let fData = new FormData(postForm);
	let req = new XMLHttpRequest();
	req.open("POST", postForm.dataset["action"], true);
	req.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.status == 200) {
				window.location.reload();
			} else {
				console.log(this.responseText);
			}
		}
	};
	req.send(fData);
});
