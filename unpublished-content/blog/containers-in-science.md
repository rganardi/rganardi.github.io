---
title: "Containers in Science"
date: 2020-03-27T21:02:55+01:00
draft: true
---

One thing that I think docker would be a huge help is in scientific computing, specifically simulations.

# Problem

Simulations usually have many dependencies, not all of the are necessarily easy to install.
Some of the are even self-written libraries, written by graduate students who share the code among themselves in a zip.
This makes it hard to reproduce simulations.
For me, this means only running simulations on laptop that's left overnight.
Having all of them in a container will definitely make running simulations much easier.

# Solution

# Flow

The flow that I have in mind is this:
 1. Build the container expecting input in folder `/input`, writing output in folder `/output`.
 2. When running the container, bind mount input and output so state is persisted.
 3. Have some script to manage all the orchestration stuff.

This workflow is inspired by [papermill](https://netflixtechblog.com/scheduling-notebooks-348e6c14cfd6).

# Architecture

There would be a scheduler who assigns work to available nodes.

# Problems with it

Here's a list of packages that I need for my simulation:
 - numpy
 - private package
 - cvxopt
 - mosek

I spent a whole day trying to build an alpine linux base image suitable for scientific work.

After fighting with `pip install numpy` because it doesn't work because alpine uses musl instead of glibc, I found out that there is a `py3-numpy` package in alpine repository.

Next package that I need is `cvxopt`.
However the same trick doesn't work because `pip3 install cvxopt` fails for various reasons.

First one is there's no compiler built-in alpine (which is actually good, since there's no reason to ship a compiler in an image for serving).
Then the compiler fails to link `lapack` and `blas`.
Finally, it fails to find a header file `umfpack.h`.

After spending some time on Google, I found [this awesome image](https://hub.docker.com/r/frolvlad/alpine-python-machinelearning/) on DockerHub that does exactly what I need.
After pruning the packages that I don't really need, I finally manage to work out what to put in the Dockerfile to build the image that I want.

Here is the end result.

```Dockerfile
FROM alpine:latest

ENV CVXOPT_BLAS_LIB=openblas
RUN apk add \
	    build-base \
	    openblas \
	    openblas-dev \
	    lapack \
	    lapack-dev \
	    suitesparse \
	    suitesparse-dev \
	    py3-numpy \
	    py3-scipy \
	    && \
	    \
	    pip3 install \
	    cvxopt \
	    --global-option=build_ext \
	    --global-option="-I/usr/include/suitesparse" \
	    && \
	    apk del	\
	    build-base  \
	    openblas-dev  \
	    lapack-dev  \
	    suitesparse-dev
```
