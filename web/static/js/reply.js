window.onload = function() {
	var replyForm = document.forms.namedItem("reply");
	replyForm.onsubmit = function() {
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
};
/*
// Script to delete comment.
var req = new XMLHttpRequest();
req.open("DELETE", "/mylife/example-post-10-c95af1eefbad/comment/delete?c_id=1");
req.onreadystatechange = function() {
	if (this.readyState == 4) {
		console.log(this.responseText);
	}
};
req.send();
*/