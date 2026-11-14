// Writes a marker file into BP, naming the package version that ran, so the
// test can tell which version was installed.
const fs = require("fs");
const path = require("path");

const packageJson = require(path.join(__dirname, "package.json"));
const data = JSON.parse(
	fs.readFileSync(path.join("data", "npm-filter", "greeting.json"), "utf8"));
fs.writeFileSync(
	path.join("BP", "npm_filter_marker.json"),
	JSON.stringify(
		{
			source: packageJson.name,
			version: packageJson.version,
			greeting: data.greeting,
		},
		null,
		"\t") + "\n");
