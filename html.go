package main

func getHtml() string {
	// have this be a string so the binaries can stand alone without a public directory with a html file
	return `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
		<title>Document</title>
		<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
	</head>
	<body style="font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif">
		<canvas id="chart"></canvas>
		<p id="status" style="color: green; font-size: 2rem; font-weight: bold"></p>
		<script>
		(async () => {
			async function getData() {
				const res = await fetch("data");
				const data = await res.json();

				if (data.done) {
					document.getElementById("status").textContent = "Done!";
				}

				return data;
			}

			const apiData = await getData();

			const data = {
				labels: apiData.xAxis,
				datasets: [
					{
						label: "Average Response Time",
						data: apiData.responseTimes,
						fill: false,
						borderColor: "rgb(255, 99, 132)",
						tension: 0.1,
					},
					{
						label: "Denied Connections",
						data: apiData.denied,
						fill: false,
						borderColor: "rgb(54, 162, 235)",
						tension: 0.1,
					},
				],
			};

			const ctx = document.getElementById("chart");

			const myChart = new Chart(ctx, {
				type: "line",
				data: data,
				options: {
					scales: {
						x: {
							title: {
								display: true,
								text: "Amount of Connections",
							},
							type: "linear",
							position: "bottom",
						},
						y: {
							beginAtZero: true,
						},
					},
				},
			});

			if (!apiData.done) {
				const interval = setInterval(async () => {
					const apiData = await getData();
					myChart.data.datasets[0].data = apiData.responseTimes;
					myChart.data.datasets[1].data = apiData.denied;
					myChart.update();

					if (apiData.done) {
					clearInterval(interval);
					}
				}, 1000);
			}
		})();
		</script>
	</body>
	</html>
	`
}
