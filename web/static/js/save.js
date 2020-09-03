function setupSave() {
	// Select all the articles that have an attribute data-save-link.
	let posts = document.querySelectorAll("article[data-save-link]");
	for (let i = posts.length - 1; i >= 0; i--) {
		let saveLink = posts[i].dataset["dataSaveLink"];
		let undoSaveLink = posts[i].dataset["dataUndoSaveLink"];

		let btn = posts[i].querySelector(".save-button");
		btn.onclick = function() {
			let saved = save.dataset["saved"];
			let link;
			let finalText;
			let finalSaved;
			if (saved == "true") {
				link = undoSaveLink;
				finalText = "Save this post";
				finalSaved = "false";
			} else {
				link = saveLink;
				finalText = "You saved this post";
				finalSaved = "true";
			}
			let req = new XMLHttpRequest();
			req.onreadystatechange = function() {
				if (this.readyState == 4) {
					if (this.status == 200) {
						btn.innerHTML = finalText;
						btn.dataset["saved"] = finalSaved;
					} else {
						console.log(this.responseText);
					}
				}
			};
			req.open("POST", link, true);
			req.send();
		};
	}
}
