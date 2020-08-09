logoutBtn = document.querySelector(".user-options button");
if (logoutBtn != undefined) {
	logoutBtn.addEventListener("click", function() {
		let req = new XMLHttpRequest();
		req.open("GET", logoutBtn.dataset["href"], true);
		req.onreadystatechange = function () {
			if (this.readyState == 4) {
				if (this.status == 200) {
					window.location.href = "/";
				} else {
					console.log(this.responseText);
				}
			}
		};
		req.send();
	});
}
