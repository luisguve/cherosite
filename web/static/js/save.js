window.addEventListener("load", function() {
	saveButtons = document.getElementsByClassName("save-button");
	for (let i = saveButtons.length - 1; i >= 0; i--) {
		let thread = saveButtons[i].parentNode.parentNode.parentNode;
		saveButtons[i].addEventListener("click", function() {
			let save = this;
			let saved = save.dataset["saved"];
			let req = new XMLHttpRequest();
			if (saved == "true") {
				// Send reques to undo save.
				req.onreadystatechange = function() {
					if (this.readyState == 4) {
						if (this.status == 200) {
							save.innerHTML = "Save this thread";
							save.dataset["saved"] = false;
						} else {
							console.log(this.responseText);
						}
					}
				};
				req.open("POST", thread.dataset["undoSaveLink"], true);
				req.send();
			} else {
				// Send reques to save.
				req.onreadystatechange = function() {
					if (this.readyState == 4) {
						if (this.status == 200) {
							save.innerHTML = "You saved this thread";
							save.dataset["saved"] = true;
							console.log("thread successfully saved")
						} else {
							console.log(this.responseText);
						}
					}
				};
				req.open("POST", thread.dataset["saveLink"], true);
				req.send();
			}
		});
	}
});
