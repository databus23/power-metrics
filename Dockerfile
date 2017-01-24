FROM scratch
ADD build/power-metrics /power-metrics
CMD ["/power-metrics"]
