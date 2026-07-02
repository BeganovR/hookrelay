import time
import requests
from prometheus_client import start_http_server, Gauge

NODES = {
    "patroni1": "http://patroni1:8008",
    "patroni2": "http://patroni2:8008",
}

g_leader   = Gauge("patroni_is_leader",             "1 if node is primary",            ["node"])
g_running  = Gauge("patroni_node_running",          "1 if postgres state is running",  ["node"])
g_timeline = Gauge("patroni_timeline",              "Current WAL timeline",            ["node"])
g_dcs_lag  = Gauge("patroni_dcs_last_seen_seconds", "Seconds since last DCS contact",  ["node"])
g_repl_lag = Gauge("patroni_replication_lag_bytes", "Replication lag bytes on leader", ["node"])


def _parse_lag(v):
    if isinstance(v, (int, float)):
        return float(v)
    try:
        return float(str(v).split()[0])
    except Exception:
        return 0.0


def collect():
    now = time.time()
    for node, url in NODES.items():
        try:
            data = requests.get(f"{url}/patroni", timeout=2).json()
            g_leader.labels(node=node).set(1 if data.get("role") == "primary" else 0)
            g_running.labels(node=node).set(1 if data.get("state") == "running" else 0)
            g_timeline.labels(node=node).set(data.get("timeline") or 0)
            dcs = data.get("dcs_last_seen") or 0
            g_dcs_lag.labels(node=node).set(now - dcs if dcs else 0)
            repls = data.get("replication") or []
            g_repl_lag.labels(node=node).set(
                sum(_parse_lag(r.get("lag", 0)) for r in repls)
            )
        except Exception:
            for g in (g_leader, g_running, g_timeline, g_dcs_lag, g_repl_lag):
                g.labels(node=node).set(0)


if __name__ == "__main__":
    start_http_server(8002)
    while True:
        collect()
        time.sleep(5)
