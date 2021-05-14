"use strict";

function $(name) { return document.getElementById(name); }

function Tag(name, attrs, body, children) {
	let elem = document.createElement(name);
	if (body) {
		elem.innerHTML = body;
	}
	for (let k in attrs) {
		elem.setAttribute(k, attrs[k]);
	}
	for (let i in children) {
		elem.appendChild(children[i]);
	}
	return elem;
}

function Text(text) {
	return document.createTextNode(text);
}

function debounce(func, cancelFunc, timeout = 200) {
	let timer;
	return (...args) => {
		cancelFunc();
		clearTimeout(timer);
		timer = setTimeout(() => { func.apply(this, args); }, timeout);
	}
}

function sendRequest(method, path, query, body, funcDone, funcError) {
	let req = new XMLHttpRequest();
	if (query) {
		let params = new URLSearchParams(query);
		path = path + "?" + params.toString();
	}
	req.open(method, path, true);
	req.onreadystatechange = function () {
		if (req.readyState == XMLHttpRequest.DONE) {
			let status = req.status;
			if (status === 0 || (status >= 200 && status < 400)) {
				if (funcDone)
					funcDone(req.responseText, status);
			} else {
				if (funcError)
					funcError(req.responseText, status);
			}
		}
	}

	if (body) {
		req.setRequestHeader("Content-Type", "application/json");
		req.send(JSON.stringify(body));
	} else {
		req.send();
	}

	return req;
}

const KEY_DOWN = 40;
const KEY_UP = 38;
const KEY_ENTER = 13;

function Autocomplete(args) {
	this.input_field = args.input;
	this.parent = this.input_field.parentNode;
	this.suggestionContainer = Tag('div', {class: 'autocomplete-items'});
	this.parent.appendChild(this.suggestionContainer);
	this.focus = -1;

	// funcFindVariants(value, funcSuccess(suggestion[] {.render}, funcFail)
	this.funcFindVariants = args.funcFindVariants;
	this.funcSelect = args.funcSelect; // (suggestion)
	this.current_search = null;

	this.input_field.addEventListener('input', (e) => {
		this.clearSuggestionList();
		let value = this.input_field.value;
		if (value) {
			this.requestSuggestionList(value);
		}
	});

	this.input_field.addEventListener('keydown', (e) => {
		if (!this.suggestions)
			return;

		if (e.keyCode == KEY_DOWN) {
			e.preventDefault();
			this.changeFocus(1);
		} else if (e.keyCode == KEY_UP) {
			e.preventDefault();
			this.changeFocus(-1);
		} else if (e.keyCode == KEY_ENTER) {
			e.preventDefault();
			if (this.focus >= 0 && this.focus < this.suggestions.length) {
				this.select(this.focus);
			}
		}
	});
}

Autocomplete.prototype.select = function(index) {
	let selection = this.suggestions[index];
	this.clearSuggestionList();
	this.input_field.blur();
	this.funcSelect(selection);
}

Autocomplete.prototype.changeFocus = function(delta) {
	let prevFocus = this.focus;
	this.focus += delta;
	if (this.focus < 0) this.focus = this.suggestions.length - 1;
	if (this.focus >= this.suggestions.length) this.focus = 0;

	if (prevFocus >= 0 && prevFocus < this.suggestions.length)
		this.suggestions[prevFocus].deactivate();
	if (this.focus >= 0 && this.focus < this.suggestions.length)
		this.suggestions[this.focus].activate();
}

Autocomplete.prototype.clearSuggestionList = function() {
	if (this.current_search) {
		this.current_search.abort();
		this.current_search = null;
	}
	this.suggestionContainer.innerHTML = "";
	this.focus = -1;
	this.suggestions = [];
}

Autocomplete.prototype.requestSuggestionList = function(value) {
	this.clearSuggestionList();
	debounce(() => {
		this.clearSuggestionList();
		this.current_search = this.funcFindVariants(value, (suggestions) => {
			this.suggestions = suggestions;
			for (let i in this.suggestions) {
				let item = this.suggestions[i].element;
				// TODO add click
				let elem = Tag('div', {class:'autocomplete-item'}, null, [item]);
				elem.addEventListener("click", (e) => {
					this.select(i);
				});
				this.suggestionContainer.appendChild(elem);
			}
		}, (error) => {
			abort("LOL ERROR", error);
		});
	}, () => {}
	)();
}
