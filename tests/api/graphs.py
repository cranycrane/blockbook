import pandas as pd
import matplotlib.pyplot as plt
from matplotlib.lines import Line2D

csv_path = "systemSeq96GB-TronImproved.csv"
df = pd.read_csv(csv_path, parse_dates=["ts"])

df["elapsed_s"] = (df["ts"] - df["ts"].iloc[0]).dt.total_seconds()

df = df.rename(columns={"cpuP": "cpuPercent", "iowaitP": "iowaitPercent"})

fig, ax_cpu = plt.subplots(figsize=(12, 5))

window = 5
df["cpuSmoothed"]    = df["cpuPercent"].rolling(window).mean()
df["iowaitSmoothed"] = df["iowaitPercent"].rolling(window).mean()

ax_cpu.plot(df["elapsed_s"], df["cpuSmoothed"], label="CPU %", color="tab:blue")
ax_cpu.set_ylabel("CPU Utilisation [%]", color="tab:blue")
ax_cpu.set_ylim(0, 100)
ax_cpu.tick_params(axis='y', labelcolor="tab:blue")

ax_io = ax_cpu.twinx()
ax_io.plot(df["elapsed_s"], df["iowaitSmoothed"], label="IO‑wait %", color="tab:green")
ax_io.set_ylabel("IO‑wait [%]", color="tab:green")
ax_io.set_ylim(0, df["iowaitSmoothed"].max() + 2.5)
ax_io.tick_params(axis='y', labelcolor="tab:green")

ax_cpu.set_xlabel("Time [s]")

if "blkRequested" in df.columns:
    for x in df.loc[df["blkRequested"] > 0, "elapsed_s"]:
        ax_cpu.axvline(x=x, color="gray", linestyle="--", linewidth=1, alpha=0.6)

    block_line = Line2D([0], [0], color="gray", linestyle="--", linewidth=1, label="Block request")
else:
    block_line = None

lines, labels = ax_cpu.get_legend_handles_labels()
lines2, labels2 = ax_io.get_legend_handles_labels()

if block_line:
    lines += [block_line]
    labels += ["Block request"]

ax_cpu.legend(lines + lines2, labels + labels2, loc="upper left")

plt.title("CPU vs IO‑wait over time")
plt.tight_layout()
plt.savefig("graphSeq96GB-TronImproved.png", dpi=150)

