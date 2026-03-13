FROM docker.elastic.co/beats/filebeat:8.14.0

USER root

COPY internal/logger/filebeat/filebeat.yml \
     /usr/share/filebeat/filebeat.yml

RUN chmod go-w /usr/share/filebeat/filebeat.yml && \
    chown root:filebeat /usr/share/filebeat/filebeat.yml

USER filebeat
