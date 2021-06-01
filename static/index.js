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
}
