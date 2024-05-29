srcs=$(wildcard *.md)
output=$(patsubst %.md,public/%.html,$(srcs))

all: $(output)

public/%.html: %.md
	pandoc -s $< -o $@

publications: public/publications.html

public/publications.html: FORCE
	python fetch-arxiv.py > $@

.PHONY: FORCE
