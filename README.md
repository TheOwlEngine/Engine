# Engine

Descriptive scraper engine based on Chromium browser

## Roadmap

- [x] scraper Engine Server (single Browser, multiple Tabs)
- [x] Command line connector
- [x] Multiple flow configuration (Yaml)
- [x] Web Extraction (HTML & Text)
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
engine: http://127.0.0.1:3000
webpage: https://google.com

# Basic Flow
flow:
  - name: Page 1
    step:

      # Test searching
      - element: .gLFyf.gsfi
        write: scraper
      - element: .gLFyf.gsfi
        action: Enter
      
      # Wait until 2 (second)
      - delay: 2

      # Remove text on search field
      - extract:
          element: .kno-rdesc
          name: sample-1
          result: html

      # Remove text on search field
      - extract:
          element: .hgKElc
          name: sample-2
          result: text

      # Remove text on search field
      - extract:
          element: .Wt5Tfe
          result: text

```

And then you can see the result of this **flow** at `resources/json` directory, with log for every process available on `logs` directory

> You can see another example **flow** on the `flows` directory

## License

This source code under MIT license