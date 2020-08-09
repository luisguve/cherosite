var articles = document.getElementsByTagName('article');
for (var i = articles.length - 1; i >= 0; i--) {
	let upvoteLink = articles[i].dataset["upvoteLink"];
	let undoUpvoteLink = articles[i].dataset["undoUpvoteLink"];
	let btn = articles[i].querySelector(".upvotes > button");
	btn.onclick = function() {
		let upvoted = btn.dataset["upvoted"];
		let req = new XMLHttpRequest();
		if (upvoted == "true") {
			// Post request to undo upvote.
			req.open("POST", undoUpvoteLink);
			req.onreadystatechange = function() {
				if (this.readyState == 4) {
					if (this.status == 200) {
						btn.dataset["upvoted"] = false;
						let upvotes = parseInt(btn.innerHTML);
						upvotes--;
						btn.innerHTML = upvotes + " Upvotes";
					} else {
						console.log(this.responseText);
					}
				}
			};
		} else {
			// Post request to upvote.
			req.open("POST", upvoteLink);
			req.onreadystatechange = function() {
				if (this.readyState == 4) {
					if (this.status == 200) {
						btn.dataset["upvoted"] = true;
						let upvotes = parseInt(btn.innerHTML);
						upvotes++;
						btn.innerHTML = upvotes + " Upvotes";
					} else {
						console.log(this.responseText);
					}
				}
			};
		}
		req.send();
	};
}