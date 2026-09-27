# ERIA Project - Weather service

Current weather from [Open-Meteo](https://open-meteo.com) (free, no API key,
non-commercial use): temperature (°C), humidity (%), pressure at sea level
(hPa), wind speed (m/s) and direction (°), UV index.

Exposed as `eria:service:weather:1`, updated every `rate` seconds (default
900: Open-Meteo refreshes its current values every 15 min). On an API error
the last values are kept and the next tick retries.

See `config.sample.yml`.
