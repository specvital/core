module.exports = {
  "**/*.{json,yml,yaml,md}": () => "just lint config",
  justfile: () => "just lint justfile",
};
