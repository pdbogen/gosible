FROM ubuntu:trusty

RUN export DEBIAN_FRONTEND=noninteractive && \
    for i in $(seq 1 5); do apt-get update && break; done; \
    for i in $(seq 1 5); do apt-get -y install openssh-server && break; done && \
    printf 'some_secret_value\nsome_secret_value\n' | passwd

RUN mkdir /var/run/sshd

ADD run.sh /run.sh
ADD sshd_config /etc/ssh/sshd_config

ENTRYPOINT /run.sh
