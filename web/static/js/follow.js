/*window.addEventListener("load", function() {
	followDivs = document.getElementsByClassName("follow");
	for (var i = followDivs.length - 1; i >= 0; i--) {
		followDivs[i]
	}
});*/
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

