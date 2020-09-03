// Only user profile pages have a button to follow/unfollow the user.
function setupFollow() {
	let follow = document.querySelector(".follow");

	let followLink = follow.dataset["followLink"];
	let unfollowLink = follow.dataset["unfollowLink"];

	// The button may have not been rendered in case of a user viewing his own
	// page. In this case, the button variable will be null and the setup will
	// do nothing but return.
	let btn = follow.querySelector("button");
	if (btn == null) {
		return;
	}

	btn.onclick = function() {
		let following = follow.dataset["following"];
		let link;
		let finalText;
		let finalFollowing;
		if (following == "true") {
			link = unfollowLink;
			finalText = "Follow";
			finalFollowing = "false";
		} else {
			link = followLink;
			finalText = "Unfollow";
			finalFollowing = "true";
		}
		let req = new XMLHttpRequest();
		req.open("POST", link, true);
		req.onreadystatechange = function() {
			if (this.readyState == 4) {
				if (this.status == 200) {
					btn.innerHTML = finalText;
					follow.dataset["following"] = finalFollowing;
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

