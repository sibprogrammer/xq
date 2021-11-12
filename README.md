# xq

Command line XML beautifier and content extractor. Similar to jq.

![xq](./assets/images/screenshot.png?raw=true)

# Features

* Syntax highlighting
* Automatic indentation
* Automatic pagination
* Node content extraction

# Usage

Format an XML file and highlight the syntax:

```
xq test/data/unformatted.xml
```

`xq` also accepts input through `stdin`:

```
curl -s https://www.w3schools.com/xml/note.xml | xq
```

It is possible to extract the content using XPath query language.
`-x` parameter accepts XPath expression.

Extract the text content of all nodes with `city` name:

```
cat test/data/unformatted.xml | xq -x //city
```

Extract the value of attribute named `status` and belonging to `user`:

```
cat test/data/unformatted.xml | xq -x /user/@status
```

See https://en.wikipedia.org/wiki/XPath for details.

# Installation

A simple way to install the utility is to use the curl and bash installer.

For macOS:
```
curl -sSL https://git.io/install-xq | bash
```

For Linux:
```
curl -sSL https://git.io/install-xq | sudo bash
```

If you have Go toolchain installed, you can use the following command to install `xq`:
```
go install github.com/sibprogrammer/xq@latest
```