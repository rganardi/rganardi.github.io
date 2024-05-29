---
title: "Make Workflow Management System"
date: 2021-12-28T22:35:53+01:00
draft: true
---

Anyone who has spent some time writing code for scientific simulation or data crunching has run into the same problem: "how to make sure the analysis is reproducible?"
While there has been [many](https://snakemake.readthedocs.io/en/stable/), [many](https://www.nextflow.io/) tools that has been written to solve this problem, I will argue that [GNU Make](https://www.gnu.org/software/make/) is the perfect tool for this task.

A workflow management system is a tool to help ensure the reproducibility of an analysis.
My basic requirements are as follows:
 - Easy to write "recipes"
 - Easy to run
 - Keeps track of which analysis is stale
 - Widely availability in different platforms

A moment's thought will reveal that these requirements are shared with software build tools, the most common and ubiquitous being GNU Make.

Therefore, I present the following Makefile for reproducible science

	# Makefile

	SRC=001-two-qubit-conjecture.py
	all-outputs:=$(patsubst 001-inputs/%.toml, 001-outputs/%.log, $(wildcard 001-inputs/*.toml))

	.PHONY: data prepare clean
	.DEFAULT_GOAL := data

	001-outputs/%.log: 001-inputs/%.toml $(SRC)
		time poetry run python $(SRC) $<

	data: $(all-outputs)

	prepare:
		poetry install
		mkdir 001-outputs

	clean:
		rm 001-outputs/*

As well as fulfilling the requirements, this setup has several important features:
 - Numbered pipelines to make sure we can trace which code produced the data from that analysis 3 years ago.
 - One input file generates one output file
 - Dependency management (here I used poetry)
 - Make takes care of managing which output is stale
 - Easy to run, just `make`.

### Python dependency management
use pipenv and pyenv/asdf to manage your python version.
*DO NOT* use the global python, that will break when your system decides to upgrade the python version.
