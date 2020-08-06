window.onload = function() {
	var replyForm = document.forms.namedItem("reply");
	var replyButton = replyForm.querySelector("button");
	replyButton.onclick = function() {
		// Set replyForm again.
		replyForm = document.forms.namedItem("reply");
		var replyLink = replyForm.dataset["action"];
		console.log(replyLink);
		var fData = new FormData(replyForm);
		var req = new XMLHttpRequest();
		req.open("POST", replyLink);
		req.onreadystatechange = function() {
			if (this.readyState == 4) {
				console.log(this.responseText);
			}
		};
		req.send(fData);
	};
	var replyComs = document.getElementsByClassName('replyCom');
	for (var i = replyComs.length - 1; i >= 0; i--) {
		replyForm = replyComs[i];
		replyButton = replyForm.querySelector("button");
		replyButton.onclick = function(i) {
			return function() {
				// Set replyform again.
				replyForm = replyComs[i];
				var replyLink = replyForm.dataset["action"];
				console.log(replyLink);
				var fData = new FormData(replyForm);
				var req = new XMLHttpRequest();
				req.open("POST", replyLink);
				req.onreadystatechange = function() {
					if (this.readyState == 4) {
						console.log(this.responseText);
					}
				};
				req.send(fData);
			};
		}(i);
	}
};
/*
// Script to delete comment.
var req = new XMLHttpRequest();
req.open("DELETE", "/mylife/example-post-16-2e1c906bc96c/comment/delete?c_id=5");
req.onreadystatechange = function() {
	if (this.readyState == 4) {
		console.log(this.responseText);
	}
};
req.send();

// Script to get 10 subcmments.
var req = new XMLHttpRequest();
req.open("GET", "/mylife/example-post-16-2e1c906bc96c/comment/?c_id=1&offset=0")
req.setRequestHeader("X-Requested-With", "XMLHttpRequest");
var response
req.onreadystatechange = function() {
	if (this.readyState == 4) {
		response = this.responseText;
	}
};
req.send();
*/
