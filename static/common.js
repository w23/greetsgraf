"use strict";

function autocompleteGroup(text_input_element, picked_func) {
	return new Autocomplete({
		input: text_input_element,
		funcSelect: (item) => {
			picked_func(item.name, item.id);
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
						let name = obj.Name;
						if (obj.Disambiguation) {
							name += " / " + obj.Disambiguation;
							desc.push(Tag("span", {class: "group"}, " / " + obj.Disambiguation));
						}
						desc.push(
							Tag("span", {class: "group"}, " prods: " + obj.ProdsCount + " greets: " + obj.GreetsCount)
						);
						let element = Tag('div', {class: "item-inactive"}, null, desc);
						suggestions.push({
							id: obj.ID,
							name: name,
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
}
