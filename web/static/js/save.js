saveButtons = document.getElementsByClassName("save-button");
for (let i = saveButtons.length - 1; i >= 0; i--) {
	let post = saveButtons[i].parentNode.parentNode.parentNode;
	saveButtons[i].addEventListener("click", function() {
		let save = this;
		let saved = save.dataset["saved"];
		let req = new XMLHttpRequest();
		if (saved == "true") {
			// Send reques to undo save.
			req.onreadystatechange = function() {
				if (this.readyState == 4) {
					if (this.status == 200) {
						save.innerHTML = "Save this post";
						save.dataset["saved"] = false;
					} else {
						console.log(this.responseText);
					}
				}
			};
			req.open("POST", post.dataset["undoSaveLink"], true);
			req.send();
		} else {
			// Send reques to save.
			req.onreadystatechange = function() {
				if (this.readyState == 4) {
					if (this.status == 200) {
						save.innerHTML = "You saved this post";
						save.dataset["saved"] = true;
						console.log("post successfully saved")
					} else {
						console.log(this.responseText);
					}
				}
			};
			req.open("POST", post.dataset["saveLink"], true);
			req.send();
		}
	});
}
