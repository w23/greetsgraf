"use strict";

function Greet(index, data) {
	this.index = index;
	this.data = data ? data : {};
}

Greet.prototype.createRow = function() {
	let tag_input_group =  Tag('input', {
		type: "text",
		value: this.data.GreeteeID
	});
	let tag_input_note = Tag('input', {
		type: "text",
		value: this.data.Reference
	});
	let tag_button_delete = Tag('button', {
		onclick: () => {
			if (confirm("Are you sure you want to delete this greet?")) { // TODO show greet info
				this.delete();
			}
		}
	}, "Delete");

	return Tag('tr', {class: "greet"}, null, [
		Tag('td', null, null, [
				Text("Group: "), tag_input_group,
				Text(" Note: "), tag_input_note,
				Text(" "), tag_button_delete
		])
	]);
};

Greet.prototype.delete = function() {
	let xhr = sendRequest("DELETE", "/v1/greets/" + this.data.ID, null, null,
		(response, status) => {
			console.log(status, response);
			loadProd();
		},
		(response, status) => {
			console.log(status, response)
			document.getElementById("greets_list").innerHTML = "ERROR:" + response;
		}
	)
}

function loadProd() {
	let pid = parseInt(document.getElementById("in_prod_id").value);
	if (isNaN(pid)) {
		return;
	}

	let prod_container = document.getElementById("prod_container");
	prod_container.style.display = "none";
	let prod = document.getElementById("prod");
	prod.innerHTML = "";

	let greets_list = document.getElementById("greets_list");
	greets_list.innerHTML = "";

	let xhr = sendRequest("GET", "/v1/prods/" + pid, null, null,
		(response) => {
			let json = JSON.parse(response);

			let html = "<strong>" + json.Name + "</strong> by ";
			for (let gi in json.Groups) {
				let g = json.Groups[gi];
				if (gi > 0) {
					html += " and ";
				}
				html += '<a href="https://www.pouet.net/groups.php?which=' + g.ID + '">' + g.Name + '</a>';
			}
			html += " <br />";
			prod.innerHTML = html;
			if (json.Screenshot) {
				prod.appendChild(Tag('img', {src: json.Screenshot}));
				prod.appendChild(Tag('br'));
			}
			prod.appendChild(Tag('a', {href: "https://www.pouet.net/prod.php?which=" + json.ID}, '[pouet.net]'));
			prod.appendChild(Text(" "));
			prod.appendChild(Tag('a', {href: "https://demozoo.org/productions/" + json.Demozoo}, '[demozoo]'));
			prod.appendChild(Text(" "));
			prod.appendChild(Tag('a', {href: "https://www.pouet.net/prod_nfo.php?which=" + json.ID}, '[nfo]'));

			if (json.Video) {
				prod.appendChild(Text(" "));
				prod.appendChild(Tag('a', {href: json.Video}, '[video]'));
			}

			for (let i in json.Greets) {
				let greet = new Greet(i, json.Greets[i]);
				greets_list.appendChild(greet.createRow());
			}

			prod_container.style.display = "block";
		},
		(response) => {
			document.getElementById("prod").innerHTML = "ERROR:" + response;
		}
	)
}
function greetAdd() {
	let pid = parseInt(document.getElementById("in_prod_id").value);
	if (isNaN(pid)) {
		return;
	}

	let gid = parseInt(document.getElementById("greet_add_group_id").value);
	if (isNaN(gid)) {
		return;
	}

	let reference = document.getElementById("greed_add_reference").value;

	let body = {
		ProdId: pid,
		GroupId: gid,
		Reference: reference
	};

	let xhr = sendRequest("POST", "/v1/greets/", null, body,
		(response, status) => {
			console.log(status, response);
			loadProd();
		},
		(response, status) => {
			console.log(status, response)
			document.getElementById("greets_list").innerHTML = "ERROR:" + response;
		}
	)
}

window.onload = function() {
	$("in_prod_id").onkeyup = (e) => {
		if (e.key === "Enter" || e.keyCode === 13) {
			loadProd();
		}
	};

	let prod_autocomplete = new Autocomplete({
		input: $('in_prod'),
		funcSelect: (item) => {
			console.log("Selected", item);
			$('in_prod_id').value = item.id;
			loadProd();
		},
		funcFindVariants: (value, found, error) => {
			let query = { name: value };
			let xhr = sendRequest("GET", "/v1/prods/search", query, null,
				(response) => {
					let json = response ? JSON.parse(response) : null;
					if (!json) {
						found([]);
						return;
					}

					let suggestions = [];

					json.forEach((obj) => {
						let element = Tag('div', {class: "item-inactive"}, null, [
							Text(obj.Name + (obj.Disambiguation ? " (" + obj.Disambiguation + ")" : "")),
							Text(" by "),
								(obj.Groups && obj.Groups.length > 0) ? Tag("span", {class: "group"}, obj.Groups[0].Name) : Text("N/A")]);
						suggestions.push({
							id: obj.ID,
							element: element,
							deactivate: () => { element.setAttribute("class", "item-inactive"); },
							activate: () => { element.setAttribute("class", "item-active"); },
						});
					});

					found(suggestions);
				},
				(response, status) => {
					error({http_status: status, reponse: response});
				}
			)
		}
	});

	let group_autocomplete = new Autocomplete({
		input: $('greet_add_group_id'),
		funcSelect: (item) => {
			$('greet_add_group_id').value = item.id;
		},
		funcFindVariants: (value, found, error) => {
			let query = { name: value };
			let xhr = sendRequest("GET", "/v1/groups/search", query, null,
				(response) => {
					let json = response ? JSON.parse(response) : null;
					if (!json) {
						found([]);
						return;
					}

					let suggestions = [];

					json.forEach((obj) => {
						let desc = [Tag("strong", null, obj.Name)];
						if (obj.Disambiguation) {
							desc.push(
								Tag("span", {class: "group"}, null, [
									Text(" / " + obj.Disambiguation)
								])
							);
						}
						let element = Tag('div', {class: "item-inactive"}, null, desc);
						suggestions.push({
							id: obj.ID,
							element: element,
							deactivate: () => { element.setAttribute("class", "item-inactive"); },
							activate: () => { element.setAttribute("class", "item-active"); },
						});
					});

					found(suggestions);
				},
				(response, status) => {
					error({http_status: status, reponse: response});
				}
			)
		}
	});
}
