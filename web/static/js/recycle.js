function Section(prev, next, contentArea, noContentArea) {
	this.pages = [];
	if (contentArea.innerHTML != "") {
		this.pages.push(contentArea.innerHTML);
	}
	this.currentPage = 0;
	this.firstPage = 0;
	this.lastPage = 0;
	this.content = contentArea;
	this.noContent = noContentArea;
	this.prevFn = function() {
		if (this.currentPage == this.firstPage) {
			alert("This is the first page");
			return;
		}
		this.currentPage--;
		this.content.innerHTML = this.pages[this.currentPage];
	};
	this.nextFn = function() {
		if (this.currentPage == this.lastPage) {
			alert("This is the last page");
			return;
		}
		this.currentPage++;
		this.content.innerHTML = this.pages[this.currentPage];
	};
	this.addPage = function(page) {
		if (page == "") {
			alert("There is no new content. Check back later.");
			return;
		}
		this.pages.push(page);
		if (this.pages.length > 1) {
			this.lastPage++;
			this.currentPage = this.lastPage;
		}
		if (this.noContent != undefined) {
			this.noContent = "";
		}
		this.content.innerHTML = page;
	}
	prev.onclick = this.prevFn;
	next.onclick = this.nextFn;
}
