FROM grafana/k6:latest

WORKDIR /home/k6

COPY ./k6/load_test.js load_test.js

ENTRYPOINT ["k6", "run", "load_test.js"]