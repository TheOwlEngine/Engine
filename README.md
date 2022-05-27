# Engine

Descriptive scraper engine based on Chromium browser

## Roadmap

- [x] Base Engine (single Browser, multiple Tabs)
- [x] CLI program to read flow(s) file
- [x] Multiple flow configuration (YAML)
- [x] Web Extraction (HTML & Text)
- [x] Add repeat & paginate parameter
- [x] Nullish / Unknown element handling
- [x] Server connection checking
- [x] Web Extraction (Table Format)
- [x] Handling environment on flow configuration
- [x] Asynchronous scraping process
- [x] Better console output
- [ ] Install from Go Package / Github
- [ ] Better project documentation

## Dependencies

- FFMPEG for Video compression

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
target: https://www.w3schools.com/css/css_table.asp

# Flow Name
name: W3Schools Table

# Flow Process
flow:

  # Get data
  - take:

      # Get heading page as heading
      - selector: h1
        name: title
        parse: text

      # Get heading page as heading
      - selector: .intro
        name: introduction
        parse: text
      
      # Get table
      - table:
          name: customer_list
          selector: "#customers"
          fields:
            - index: 1
              name: company
            - index: 0
              name: country
            - index: 2
              name: contact
```

Scraper result

```json
{
    "id": "8498d58104d5",
    "code": 200,
    "name": "W3Schools Table",
    "target": "https://www.w3schools.com/css/css_table.asp",
    "engine": "http://127.0.0.1:3000",
    "html": {
        "0": {
            "customer_list": [
                {
                    "company": "Alfreds Futterkiste",
                    "contact": "Germany",
                    "country": "Maria Anders"
                },
                ...
            ],
            "introduction": "The look of an HTML table can be greatly improved with CSS:",
            "title": "CSS Tables"
        }
    }
}
```

You can see a detailed result of this **flow** at `resources/json` directory, with log for every process available on `logs` directory.

> You can see another example on the `flows` directory

## License

This source code under MIT license