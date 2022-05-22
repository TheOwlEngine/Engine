# Engine

Descriptive scraper engine based on Chromium browser

## Roadmap

- [x] Scraper Engine Server (single Browser, multiple Tabs)
- [x] Command line connector
- [x] Multiple flow configuration (Yaml)
- [x] Web Extraction (HTML & Text)
- [x] Add repeat & paginate parameter
- [x] Nullish / Unknown element handling
- [ ] Web Extraction (Table & Custom Format)
- [ ] Better console output
- [ ] Support Windows architecture
- [ ] Install from Go Package / Github
- [ ] Parallel Tabs Processing
- [ ] Better network connection checking
- [ ] Better project documentation
- [ ] Dedicated Engine Server
- [ ] And more

## Installation

Fork & clone this repository

```
$ git clone git@github.com:your_username/Engine.git
```
Run Engine Server
```
$ go run main.go
```
Run CLI Connector
```
$ go run bin/cli.go
```

## Usage

In below example we'll search keyword **scraper** on Google and extract the content to JSON

```yaml
# Engine Server
engine: http://127.0.0.1:3000

# Target Website
target: https://en.wikipedia.org/wiki/Zebra

# Flow Name
name: Find Zebra

# Flow Process
flow:

  # Get data
  - take:

      # Get heading page as heading
      - selector: "#firstHeading"
        name: heading
        parse: text

      # Test add unknown element
      - next_to_contains:
          selector: "td"
          text: ".thumb.tright"
        name: description
        parse: text

  # Get data
  - take:

      # Get value next to kingdom element
      - next_to_contains:
          selector: "td"
          text: "Kingdom:"
        name: kingdom
        parse: text

      # Get value next to phylum element
      - next_to_contains:
          selector: "td"
          text: "Phylum:"
        name: phylum
        parse: text

      # Get value next to class element
      - next_to_contains:
          selector: "td"
          text: "Class:"
        name: class
        parse: text

  # Finding anchor with text "Fauna of Africa" and click it
  - navigate: "Fauna of Africa"

  # Get data
  - take:

      # Get heading page as sub heading
      - selector: "#firstHeading"
        name: sub_heading
        parse: text

      # Test add unknown element
      - next_to_contains:
          selector: "td"
          text: ".thumb.tright"
        name: sub_description
        parse: text
```

And then you can see the result of this **flow** at `resources/json` directory, with log for every process available on `logs` directory

> You can see another example **flow** on the `flows` directory

## License

This source code under MIT license