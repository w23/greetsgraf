"use strict";

let greetAdd;

function Greet(index, data) {
	this.index = index;
	this.data = data ? data : {};
}

Greet.prototype.createRow = function() {
	let tag_input_group =  Tag('input', {
		type: "text",
		value: this.data.Group.Name
	});
	if (this.data.Group.Disambiguation) {
		tag_input_group.value += " / " + this.data.Group.Disambiguation;
	}
	tag_input_group.disabled = true;

	let tag_input_note = Tag('input', {
		type: "text",
		value: this.data.Note
	});
	tag_input_note.disabled = true;

	let tag_button_delete = Tag('button', null, "Delete");
	tag_button_delete.onclick = () => {
		if (confirm("Are you sure you want to delete this greet?")) { // TODO show greet info
			this.delete();
		}
	};

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

function loadProd(args) {
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
			if (json.Demozoo > 0) {
				prod.appendChild(Tag('a', {href: "https://demozoo.org/productions/" + json.Demozoo}, '[demozoo]'));
			}
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

			if (args && args.clear_greets) {
				greetAdd.clear();
			}
		},
		(response) => {
			document.getElementById("prod").innerHTML = "ERROR:" + response;
		}
	)
}


window.onload = function() {
	greetAdd = function () {
		let self = Object();

		let greet_add_group = $('greet_add_group');
		let greet_add_note = $('greet_add_note');
		let greet_add_button = $('greet_add_button');
		let greet_add_group_id = null;

		self.clear = () => {
			greet_add_group.value = "";
			greet_add_note.value = "";
			greet_add_button.disabled = true;
			greet_add_group_id = null;
		}

		self.clear();

		self.add = () => {
			let pid = parseInt($("in_prod_id").value);
			if (isNaN(pid)) {
				return;
			}

			if (!greet_add_group_id || isNaN(greet_add_group_id)) {
				return;
			}

			let note = greet_add_note.value;

			let body = {
				ProdId: pid,
				GroupId: greet_add_group_id,
				Note: note
			};

			let xhr = sendRequest("POST", "/v1/greets/", null, body,
				(response, status) => {
					console.log(status, response);
					self.clear();
					loadProd();
				},
				(response, status) => {
					console.log(status, response)
					alert("ERROR:" + response);
				}
			)
		}

		greet_add_button.onclick = self.add;

		self.setGroupNameAndId = (name, gid) => {
			greet_add_group.value = name;
			greet_add_group_id = gid;
			greet_add_button.disabled = false;
		}

		return self;
	}();

	$("in_prod_id").onkeyup = (e) => {
		if (e.keyCode === KEY_ENTER || e.key === "Enter") {
			loadProd({clear_greets: true});
		}
	};

	let prod_autocomplete = new Autocomplete({
		input: $('in_prod'),
		funcSelect: (item) => {
			console.log("Selected", item);
			$('in_prod').value = item.name;
			$('in_prod_id').value = item.id;
			loadProd({clear_greets: true});
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
							name: obj.Name,
							element: element,
							deactivate: () => { element.setAttribute("class", "item-inactive"); },
							activate: () => { element.setAttribute("class", "item-active"); },
						});
					});

					found(suggestions);
				},
				(response, status) => {
					error({http_status: status, reponse: JSON.parse(response)});
				}
			)
			return xhr;
		}
	});

	let group_autocomplete = autocompleteGroup($('greet_add_group'),
		(name, id) => {
			greetAdd.setGroupNameAndId(name, id);
		}
	);
}
