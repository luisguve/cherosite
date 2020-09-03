function setupUpvotes() {
	// Select all the articles that have an attribute data-upvote-link.
	let posts = document.querySelectorAll("article[data-upvote-link]");
	for (let i = posts.length - 1; i >= 0; i--) {
		let upvoteLink = posts[i].dataset["upvoteLink"];
		let undoUpvoteLink = posts[i].dataset["undoUpvoteLink"];

		let btn = posts[i].querySelector(".upvotes > button");
		btn.onclick = function() {
			let upvoted = btn.dataset["upvoted"];
			let link;
			let finalUpvoted;
			let update;

			if (upvoted == "true") {
				link = undoUpvoteLink;
				finalUpvoted = "false";
				update = function(numUpvotes) {
					return numUpvotes - 1;
				};
			} else {
				link = upvoteLink;
				finalUpvoted = "true";
				update = function(numUpvotes) {
					return numUpvotes + 1;
				};
			}

			let req = new XMLHttpRequest();
			req.open("POST", link, true);
			req.onreadystatechange = function() {
				if (this.readyState == 4) {
					if (this.status == 200) {
						btn.dataset["upvoted"] = finalUpvoted;
						let upvotes = parseInt(btn.innerHTML);
						upvotes = update(upvotes);
						btn.innerHTML = upvotes + " Upvotes";
					} else {
						console.log(this.responseText);
					}
				}
			};
			req.send();
		};
	}
}