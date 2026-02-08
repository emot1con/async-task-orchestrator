FROM docker.elastic.co/logstash/logstash:8.11.0

USER root

RUN rm -f /usr/share/logstash/pipeline/logstash.conf && \
    rm -f /usr/share/logstash/config/logstash.yml

COPY internal/logger/logstash/config/logstash.yml \
     /usr/share/logstash/config/logstash.yml

COPY internal/logger/logstash/pipeline/logstash.conf \
     /usr/share/logstash/pipeline/logstash.conf

RUN chmod 644 /usr/share/logstash/pipeline/logstash.conf && \
    chmod 644 /usr/share/logstash/config/logstash.yml && \
    chown logstash:root /usr/share/logstash/pipeline/logstash.conf && \
    chown logstash:root /usr/share/logstash/config/logstash.yml

USER logstash
