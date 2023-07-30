# xq

[![build](https://github.com/sibprogrammer/xq/workflows/build/badge.svg)](https://github.com/sibprogrammer/xq/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/sibprogrammer/xq)](https://goreportcard.com/report/github.com/sibprogrammer/xq)
[![Codecov](https://codecov.io/gh/sibprogrammer/xq/branch/master/graph/badge.svg?token=G6QX77SQOH)](https://codecov.io/gh/sibprogrammer/xq)
[![Scc Count](https://sloc.xyz/github/sibprogrammer/xq/)](https://github.com/sibprogrammer/xq/)
[![Homebrew](https://img.shields.io/badge/dynamic/json.svg?url=https://formulae.brew.sh/api/formula/xq.json&query=$.versions.stable&label=homebrew)](https://formulae.brew.sh/formula/xq)
[![Macports](https://repology.org/badge/version-for-repo/macports/xq-sibprogrammer.svg)](https://repology.org/project/xq-sibprogrammer/versions)

Command-line XML and HTML beautifier and content extractor.

![xq](./assets/images/screenshot.png?raw=true)

# Features

* Syntax highlighting
* Automatic indentation and formatting
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

HTML content can be formatted and highlighted as well (using `-m` flag):

```
xq -m test/data/html/formatted.html
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

It is possible to use CSS selector to extract the content as well:

```
cat test/data/html/unformatted.html | xq -q "body > p"
```

Extract an attribute value instead of node content additional option `--attr` (`-a`) can be used:

```
cat test/data/html/unformatted.html | xq -q "head > script" -a "src"
```

Extract part of HTML with tags (not only text content) using CSS selector:

```
cat test/data/html/unformatted.html | xq -n -q "head"
```

# Installation

The preferable ways to install the utility are described below.

For macOS, via [Homebrew](https://brew.sh):
```
brew install xq
```

For macOS, via [MacPorts](https://www.macports.org):
```
sudo port install xq
```

For Linux using custom installer:
```
curl -sSL https://bit.ly/install-xq | sudo bash
```

For Ubuntu 22.10 or higher via package manager:
```
apt-get install xq
```

For Fedora via package manager:
```
dnf install xq
```

If you have Go toolchain installed, you can use the following command to install `xq`:
```
go install github.com/sibprogrammer/xq@latest
```
