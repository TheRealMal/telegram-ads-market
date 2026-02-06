FROM golang:1.24.0 AS builder

# Install dev/debug packages
RUN apt-get update && apt-get install -y \
  tree \
  man \
  less \
  sudo \
  vim \
  redis-tools \
  postgresql-client \
  jq \
  tmux \
  locales \
  unzip \
  zip

# Install golangci, binary will be $(go env GOPATH)/bin/golangci-lint
RUN curl -sSfL \
	https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
	sh -s -- -b $(go env GOPATH)/bin v1.59.0

# Create a `sticky` user account.
RUN useradd -m -G sudo -p $(openssl passwd -5 not) not

# Read the name of the container from the passed in argument
ARG NAME=not-dev

# Create the work dir
RUN mkdir -p /app && chown not:not /app

# Set a distinct prompt and configs
RUN echo \
  "PS1='\[\033[33m\]\u\[\033[0m\]@\[\033[33m\]$NAME\[\033[0m\]-\h:\w\$ '" \
      >> /home/not/.bashrc \
   && echo "set-option -g default-shell \"/bin/bash\"" >> /home/not/.tmux.conf

# Symlink Go binaries to /bin so tmux can find them
RUN ln -s /usr/local/go/bin/go /bin/
RUN ln -s /usr/local/go/bin/gofmt /bin/
RUN ln -s /go/bin/golangci-lint /bin

# Set the locale in order to get the correct UTF-8 output in the terminal
RUN sed -i -e 's/# \(en_US\.UTF-8 .*\)/\1/' /etc/locale.gen \
    && dpkg-reconfigure --frontend=noninteractive locales \
    && update-locale LANG=en_US.UTF-8

ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

# Allow the regular user to update certificates inside the container in order
# to allow domain spoofing without user interaction
RUN chown :not /usr/local/share/ca-certificates \
    && chmod 775 /usr/local/share/ca-certificates \
    && echo 'not ALL=(ALL:ALL) NOPASSWD: /usr/sbin/update-ca-certificates' \
        | EDITOR='tee -a' visudo

# Switch user
USER not

# Set the work dir
WORKDIR /app

# Run shell
CMD /bin/bash
