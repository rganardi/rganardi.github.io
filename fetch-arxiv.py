from bs4 import BeautifulSoup as bs
import requests

author_whitelist = [
    'Ray Ganardi',
    'Ray F. Ganardi',
]

arxiv_blacklist = [
        '2302.08120v1'
]

arxiv_api_url = 'http://export.arxiv.org/api/query?search_query=au:Ganardi_Ray&max_results=100'

header="""
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" lang="" xml:lang="">
<head>
  <meta charset="utf-8" />
  <meta name="generator" content="pandoc" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=yes" />
  <title>Ray Ganardi – Publications</title>
  <style>
    code{white-space: pre-wrap;}
    span.smallcaps{font-variant: small-caps;}
    div.columns{display: flex; gap: min(4vw, 1.5em);}
    div.column{flex: auto; overflow-x: auto;}
    div.hanging-indent{margin-left: 1.5em; text-indent: -1.5em;}
    /* The extra [class] is a hack that increases specificity enough to
       override a similar rule in reveal.js */
    ul.task-list[class]{list-style: none;}
    ul.task-list li input[type="checkbox"] {
      font-size: inherit;
      width: 0.8em;
      margin: 0 0.8em 0.2em -1.6em;
      vertical-align: middle;
    }
    .display.math{display: block; text-align: center; margin: 0.5rem auto;}
  </style>
  <p><link rel="stylesheet" href="main.css"></p>
</head>
<body>
<header id="title-block-header">
<h1 class="title">Publications</h1>

<p>
All of my publications can also be found in <a href="https://arxiv.org/search/?searchtype=author&query=Ganardi%2C+R">arxiv</a>.
</p>

<ul>
"""

footer="""
</ul>
</body>
</html>
"""

class Entry:
    def __init__(self, **kwargs):
        self.__dict__ = kwargs


    def to_html(self):
        lines = []

        lines.append('<p><li>')
        lines.append(f'{self.title}')
        lines.append(f'<div>{self.authors}</div>')

        if self.journal_ref is not None:
            if self.doi is None:
                lines.append(f'<div>{self.journal_ref}</div>')
            else:
                lines.append(f'<div><a href="https://doi.org/{self.doi}">{self.journal_ref}</a></div>')

        lines.append(f'<div><a href="https://arxiv.org/abs/{self.arxiv_id}">arxiv:{self.arxiv_id}</a></div>')
        lines.append('</p></li>')

        return '\n'.join(lines)



def emit_html(items):
    items.sort(key=lambda e: e.arxiv_id, reverse=True)

    print(header)
    for i in items:
        print(i.to_html())
    print(footer)


def process_xml(xml):
    soup = bs(xml, 'xml')

    items = []

    for e in soup.find_all('entry'):
        title = e.find('title').text
        title = title.replace('\n ', '')
        authors = ', '.join([
            a.text.rstrip().lstrip()
            for a in e.find_all('author')
        ])

        if all([a not in authors for a in author_whitelist]):
            continue

        arxiv_id = e.find('id').text[21:]
        if arxiv_id in arxiv_blacklist:
            continue

        doi = None
        if e.find('arxiv:doi') is not None:
            doi = e.find('arxiv:doi').text

        journal_ref = None
        if e.find('arxiv:journal_ref') is not None:
            journal_ref = e.find('arxiv:journal_ref').text

        items.append(Entry(
            title=title,
            authors=authors,
            arxiv_id=arxiv_id,
            doi=doi,
            journal_ref=journal_ref,
            ))


    return items


def query_arxiv():
    r = requests.get(arxiv_api_url)
    return r.text


def main():
    xml = query_arxiv()
    items = process_xml(xml)
    emit_html(items)


if __name__ == '__main__':
    main()
