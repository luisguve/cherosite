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
	var section = this;
	prev.onclick = function() {
		if (section.currentPage == section.firstPage) {
			alert("This is the first page");
			return;
		}
		section.currentPage--;
		section.content.innerHTML = section.pages[section.currentPage];
	};
	next.onclick = function() {
		if (section.currentPage == section.lastPage) {
			alert("This is the last page");
			return;
		}
		section.currentPage++;
		section.content.innerHTML = section.pages[section.currentPage];
	};
}
