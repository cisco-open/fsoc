module github.com/cisco-open/fsoc

go 1.19

require (
	github.com/apex/log v1.9.0
	github.com/briandowns/spinner v1.22.0
	github.com/charmbracelet/lipgloss v0.6.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/muesli/termenv v0.14.0
	github.com/peterhellberg/link v1.2.0
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/relvacode/iso8601 v1.3.0
	github.com/spf13/afero v1.9.4
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.15.0
	github.com/stretchr/testify v1.8.1
	github.com/xeipuuv/gojsonschema v1.2.0
	go.pinniped.dev v0.22.0
	golang.org/x/exp v0.0.0-20230304125523-9ff063c70017
	golang.org/x/oauth2 v0.5.0
	gopkg.in/yaml.v3 v3.0.1
)

// temporary fork of spinner to deal with incorrect terminal check (issue #142, PR#149)
replace github.com/briandowns/spinner v1.22.0 => github.com/pnickolov/spinner v1.22.3

require (
	github.com/aymanbagabas/go-osc52 v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	golang.org/x/term v0.5.0 // indirect
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/fatih/color v1.14.1
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/gojq v0.12.12
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/olekukonko/tablewriter v0.0.5
	github.com/pelletier/go-toml/v2 v2.0.7 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)
