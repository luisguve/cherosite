window.addEventListener("load", function() {
	var signinForm = document.forms.namedItem("signin");
	var submitBtn = signinForm.querySelector("button");
	submitBtn.addEventListener("click", function() {
		let fData = new FormData(signinForm);
		let req = new XMLHttpRequest();
		req.open("POST", signinForm.dataset["action"], true);
		req.onreadystatechange = function() {
			if (this.readyState == 4) {
				if (this.status == 200) {
					window.location.href = "/";
				} else {
					console.log(this.responseText);
				}
			}
		};
		req.send(fData);
	});
	var loginForm = document.forms.namedItem("login");
	loginForm.addEventListener("submit", function(e) {
		e.preventDefault();
		let fData = new FormData(loginForm);
		let req = new XMLHttpRequest();
		req.open("POST", loginForm.dataset["action"], true);
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
});
