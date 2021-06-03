"use strict";

function makeStatLinePct(title, have, total) {
	let pct = have * 100.0 / total;
	return Tag('div', null, null, [
		Tag('strong', null, title),
		Text(have + ' of ' + total + ' (' + pct.toFixed(3) + '%)')
	]);
}

window.onload = function() {
	sendRequest("GET", "/v1/stats", null, null,
		(response, status) => {
			let stats = $('stats');
			let json = JSON.parse(response);
			let prods_pct = json['ProdsWithGreets'] * 100.0 / json['TotalProds']
			stats.appendChild(makeStatLinePct("Prods with greets: ", json['ProdsWithGreets'], json['TotalProds']));
			stats.appendChild(makeStatLinePct("Groups greeted: ", json['GreetedGroups'], json['TotalGroups']));
			stats.appendChild(
				Tag('div', null, null, [
					Tag('strong', null, 'Total greets issued: '),
					Text(json['TotalGreets'])
				])
			);
		},
		(response, status) => {
			console.log(status, response)
		}
	);

	sendRequest("GET", "/v1/groups/greeted?limit=16", null, null,
		(response, status) => {
			let json = JSON.parse(response);

			let table = Tag('table');
			table.appendChild(Tag('tr', null, null, [
				Tag('th', null, 'Greeted'),
				Tag('th', null, 'Group name'),
			]));

			for (let i in json) {
				table.appendChild(Tag('tr', {class: "greet"}, null, [
					Tag('td', null, json[i]['count']),
					Tag('td', null, json[i]['group_name']),
				]));
			}

			$('most-greeted-groups').appendChild(table);
		},
		(response, status) => {
			console.log(status, response)
		}
	);
	let group_search = $('group-search');
	let group_search_xhr = null;
	let group_greets_table = $('group-greets-table');
	let group_autocomplete = autocompleteGroup(group_search,
		(name, id) => {
			group_search.value = name;

			if (group_search_xhr != null) {
				group_search_xhr.abort();
			}

			group_greets_table.innerHTML = "";

			group_search_xhr = sendRequest("GET", "/v1/groups/"+id+"/greets", null, null,
				(response, status) => {
					group_search_xhr = null;

					let json = JSON.parse(response);

					let table = Tag('table');
					table.appendChild(Tag('tr', null, null, [
						Tag('th', null, 'Prod'),
						Tag('th', null, 'Note'),
						Tag('th', null, 'Prod makers'),
					]));

					for (let i in json) {
						let prod = json[i]['Prod'];
						let groups = [];
						for (let j in prod['Groups']) {
							if (groups != "") {
								groups.push(Text(", "));
							}
							groups.push(
								Tag('a',
									{href: "https://www.pouet.net/groups.php?which=" + prod['Groups'][j]['ID'],},
									prod['Groups'][j]['Name'])
							);
						}
						table.appendChild(Tag('tr', {class: "greet"}, null, [
							Tag('td', null, null, [
								Text(prod['Name'] + " "),
								Tag('a', {href: "https://www.pouet.net/prod.php?which=" + prod['ID']}, '[pouet.net]'),
							]),
							Tag('td', null, json[i]['Reference']),
							Tag('td', null, null, groups),
						]));
					}

					group_greets_table.appendChild(table);
					group_greets_table.scrollIntoView(false);
				},
				(response, status) => {
					console.log(status, response)
				}
			);

		}
	);
}
