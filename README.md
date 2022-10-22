# xq

[![build](https://github.com/sibprogrammer/xq/workflows/build/badge.svg)](https://github.com/sibprogrammer/xq/actions)

Command-line XML beautifier and content extractor. Similar to jq.

![xq](./assets/images/screenshot.png?raw=true)

# Features

* Syntax highlighting
* Automatic indentation
* Automatic pagination
* Node content extraction

# Usage

Format an XML file and highlight the syntax:

```
xq test/data/xml/unformatted.xml
```

`xq` also accepts input through `stdin`:

```
curl -s https://www.w3schools.com/xml/note.xml | xq
```

It is possible to extract the content using XPath query language.
`-x` parameter accepts XPath expression.

Extract the text content of all nodes with `city` name:

```
cat test/data/xml/unformatted.xml | xq -x //city
```

Extract the value of attribute named `status` and belonging to `user`:

```
cat test/data/xml/unformatted.xml | xq -x /user/@status
```

See https://en.wikipedia.org/wiki/XPath for details.

# Installation

The preferable ways to install the utility are described below.

For macOS:
```
brew install xq
```

For Linux:
```
curl -sSL https://bit.ly/install-xq | sudo bash
```

For Fedora via package manager:
```
dnf install xq
```

If you have Go toolchain installed, you can use the following command to install `xq`:
```
go install github.com/sibprogrammer/xq@latest
```
