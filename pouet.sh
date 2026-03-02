#!/bin/bash
set -eux

DEST="./tmp"

downloadLatest() {
	local LATEST=$(mktemp)
	curl https://data.pouet.net/json.php -o "$LATEST"
	local PRODS_URL=$(jq -r .latest.prods.url "$LATEST")
	local GROUPS_URL=$(jq -r .latest.groups.url "$LATEST")

	local PRODS_EXT="${PRODS_URL##*.}"
	local GROUPS_EXT="${PRODS_URL##*.}"

	local PRODS_FILE="$DEST/pouet-latest-prods.json.$PRODS_EXT"
	local GROUPS_FILE="$DEST/pouet-latest-groups.json.$GROUPS_EXT"

	wget "$PRODS_URL" -O "$PRODS_FILE"
	wget "$GROUPS_URL" -O "$GROUPS_FILE"
}

extractTestData() {
	local PRODS_FILE="$DEST/pouet-latest-prods.json*"
	local GROUPS_FILE="$DEST/pouet-latest-groups.json*"

	local TEST_GROUPS_FILE="./test/pouet-groups.json.gz"
	local TEST_PRODS_FILE="./test/pouet-prods.json.gz"

	extractPopularGroups "$GROUPS_FILE" "$TEST_GROUPS_FILE"
	extractPopularGroupsProds "$PRODS_FILE" "$TEST_PRODS_FILE"
}

# Taken from top pouet prods
# 322 = farbrausch
# 697 = rgba
# 1623 = tbc
# 2 = exceed
# 1317 = asd
# 7439 = orb
# 196 = andromeda
# 1 = the black lotus

GROUP_IDS='.id == "322" or .id == "697" or .id == "1623" or .id == "2" or .id == "1317" or .id == "7439" or .id == "196" or .id == "1"'

extractPopularGroups() {
	# less automaticall extracts any archives
	less $1 \
		| jq -c '.groups |= map(select('"$GROUP_IDS"'))' \
		| gzip -9 > "$2"
}

extractPopularGroupsProds() {
	# less automaticall extracts any archives
	less $1 \
		| jq -c '.prods |= map(select(any(.groups[]; '"$GROUP_IDS"')))' \
		| gzip -9 > "$2"
}

"$@"
