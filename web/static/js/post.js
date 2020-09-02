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

/*
var req = new XMLHttpRequest();
var res;
req.open("GET", "/recycleactivity", true);
req.onreadystatechange = function() {
	if (this.readyState == 4) {
		if (this.status == 200) {
			res = this.responseText;
		} else {
			console.log(this.responseText);
		}
	}
};
req.setRequestHeader("X-Requested-With", "XMLHttpRequest");
req.send();
console.log(res);
*/
