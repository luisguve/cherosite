function setupFollow() {
	var follow = document.querySelector(".follow");
	var followLink = follow.dataset["follow-link"];
	var unfollowLink = follow.dataset["unfollow-link"];
	var followBtn = follow.querySelector("button");
	if (followBtn == null) {
		return;
	}
	followBtn.onclick = function() {
		let following = bool(follow.dataset["following"]);
		let link;
		let finalText;
		if (following) {
			link = followLink;
			finalText = "Follow";
		} else {
			link = unfollowLink;
			finalText = "Unfollow";
		}
		let req = new XMLHttpRequest();
		req.open("POST", link, true);
		req.onreadystatechange = function() {
			if (this.readyState == 4) {
				if (this.status == 200) {
					followBtn.innerHTML = finalText;
					follow.dataset["following"] = !following;
				} else {
					console.log(this.responseText);
				}
			}
		};
		req.send();
	};
}

// Script to execute from the console.
/*
var req = new XMLHttpRequest();
req.open("POST", "/follow?username=arodseth");
req.onreadystatechange = function() {
	if (this.readyState == 4) {
		if (this.status == 200) {
			console.log("success");
		} else {
			console.log(this.responseText);
		}
	}
};
req.send();

var usernames = [
	"bep",
	"billgates",
	"cerlant",
	"cheesetris21",
	"ct",
	"dirlewanger",
	"helloWorld",
	"hpittier",
	"johndoe",
	"luisguve",
	"m_scott",
	"mcleod",
	"mrRobot",
	"orlando",
	"packer",
	"schwarzenegger",
	"theRealDonaldTrump",
]
for (var i = 0; i < usernames.length; i++) {
	var req = new XMLHttpRequest();
	req.open("POST", "/follow?username=" + usernames[i]);
	req.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.status == 200) {
				console.log("success");
			} else {
				console.log(this.responseText);
			}
		}
	};
	req.send();
}
*/

