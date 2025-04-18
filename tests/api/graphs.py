import pandas as pd
import matplotlib.pyplot as plt

csv_path = "system.csv"
df = pd.read_csv(csv_path, parse_dates=["ts"])

df["elapsed_ms"] = (df["ts"] - df["ts"].iloc[0]).dt.total_seconds() * 1000

df = df.rename(columns={"cpuP": "cpuPercent", "iowaitP": "iowaitPercent"})

fig, ax_cpu = plt.subplots(figsize=(12, 5))

ax_cpu.plot(df["elapsed_ms"], df["cpuPercent"], label="CPU %", color="tab:blue")
ax_cpu.set_ylabel("CPU Utilisation [%]", color="tab:blue")
ax_cpu.set_ylim(0, 100)
ax_cpu.tick_params(axis='y', labelcolor="tab:blue")

ax_io = ax_cpu.twinx()
ax_io.plot(df["elapsed_ms"], df["iowaitPercent"], label="IO‑wait %", color="tab:green")
ax_io.set_ylabel("IO‑wait [%]", color="tab:green")
ax_io.set_ylim(0, 20)
ax_io.tick_params(axis='y', labelcolor="tab:green")

ax_cpu.set_xlabel("Time [ms]")

lines, labels = ax_cpu.get_legend_handles_labels()
lines2, labels2 = ax_io.get_legend_handles_labels()
ax_cpu.legend(lines + lines2, labels + labels2, loc="upper left")

plt.title("CPU vs IO‑wait over time")
plt.tight_layout()
plt.savefig("graph.png", dpi=150)

