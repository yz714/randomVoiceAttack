console.log('JavaScript loaded');

let lowFreqChart, volumeChart, fullLowFreqChart;
let fullTimestamps = [];
let selectedTimeRangeMinutes = 1;
const MAX_POINTS_FOR_CHART = 500;

function showToast(message, type = 'info') {
	const toastContainer = document.getElementById('toast-container');
	const toast = document.createElement('div');
	toast.className = `toast toast-${type}`;
	toast.textContent = message;
	toastContainer.appendChild(toast);

	setTimeout(() => {
		toast.animation = 'slideOut 0.3s ease-out forwards';
		setTimeout(() => {
			toast.remove();
		}, 300);
	}, 3000);
}

function downsampleData(labels, data, maxPoints) {
	if (labels.length <= maxPoints) {
		return { labels, data };
	}

	const resultLabels = [];
	const resultData = [];
	const step = Math.ceil(labels.length / maxPoints);

	for (let i = 0; i < labels.length; i += step) {
		let sum = 0;
		let count = 0;
		const end = Math.min(i + step, labels.length);
		
		for (let j = i; j < end; j++) {
			sum += data[j];
			count++;
		}
		
		resultLabels.push(labels[i]);
		resultData.push(sum / count);
	}

	return { labels: resultLabels, data: resultData };
}

function playRandomAudio() {
	console.log('playRandomAudio called');
	fetch('/api/audio/play/random', {
		method: 'POST'
	})
	.then(response => {
		console.log('Response received:', response);
		return response.json();
	})
	.then(data => {
		console.log('Response data:', data);
		if (data.success) {
			showToast('正在播放随机音频...', 'success');
		} else {
			showToast('播放失败: ' + data.error, 'error');
		}
	})
	.catch(error => {
		console.error('Error playing random audio:', error);
		showToast('播放失败，请检查服务器状态', 'error');
	});
}

function playAudioSequence() {
	fetch('/api/audio/play/sequence', {
		method: 'POST'
	})
	.then(response => response.json())
	.then(data => {
		if (data.success) {
			showToast('正在连续播放3次音频...', 'success');
		} else {
			showToast('播放失败: ' + data.error, 'error');
		}
	})
	.catch(error => {
		console.error('Error playing audio sequence:', error);
		showToast('播放失败，请检查服务器状态', 'error');
	});
}

function stopAudio() {
	fetch('/api/audio/stop', {
		method: 'POST'
	})
	.then(response => response.json())
	.then(data => {
		if (data.success) {
			showToast('已停止播放', 'success');
		} else {
			showToast('停止失败: ' + data.error, 'error');
		}
	})
	.catch(error => {
		console.error('Error stopping audio:', error);
		showToast('停止失败，请检查服务器状态', 'error');
	});
}

function initCharts() {
	const lowFreqCtx = document.getElementById('lowFreqChart').getContext('2d');
	lowFreqChart = new Chart(lowFreqCtx, {
		type: 'bar',
		data: {
			labels: [],
			datasets: [{
				label: '低频比率',
				data: [],
				backgroundColor: 'rgba(75, 192, 192, 0.6)',
				borderColor: 'rgba(75, 192, 192, 1)',
				borderWidth: 1
			}]
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			scales: {
				x: {
					ticks: {
						maxTicksLimit: 8,
						maxRotation: 0,
						minRotation: 0
					}
				},
				y: {
					beginAtZero: true,
					max: 1
				}
			},
			title: {
				display: true,
				text: '低频比率直方图'
			},
			plugins: {
				tooltip: {
					callbacks: {
						title: function(context) {
							const index = context[0].dataIndex;
							return fullTimestamps[index] || context[0].label;
						}
					}
				}
			}
		}
	});

	const volumeCtx = document.getElementById('volumeChart').getContext('2d');
	volumeChart = new Chart(volumeCtx, {
		type: 'line',
		data: {
			labels: [],
			datasets: [{
				label: '音量',
				data: [],
				backgroundColor: 'rgba(153, 102, 255, 0.2)',
				borderColor: 'rgba(153, 102, 255, 1)',
				borderWidth: 2,
				fill: true,
				tension: 0.4
			}]
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			scales: {
				x: {
					ticks: {
						maxTicksLimit: 8,
						maxRotation: 0,
						minRotation: 0
					}
				},
				y: {
					beginAtZero: true
				}
			},
			title: {
				display: true,
				text: '音量曲线'
			},
			plugins: {
				tooltip: {
					callbacks: {
						title: function(context) {
							const index = context[0].dataIndex;
							return fullTimestamps[index] || context[0].label;
						}
					}
				}
			}
		}
	});

	const fullLowFreqCtx = document.getElementById('fullLowFreqChart').getContext('2d');
	fullLowFreqChart = new Chart(fullLowFreqCtx, {
		type: 'line',
		data: {
			labels: [],
			datasets: [{
				label: '',
				data: [],
				backgroundColor: 'rgba(255, 159, 64, 0.2)',
				borderColor: 'rgba(255, 159, 64, 1)',
				borderWidth: 1.5,
				fill: true,
				tension: 0.4,
				pointRadius: 0
			}]
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			scales: {
				x: {
					ticks: {
						maxTicksLimit: 6,
						maxRotation: 0,
						minRotation: 0,
						font: {
							size: 9
						}
					},
					grid: {
						display: false
					}
				},
				y: {
					beginAtZero: true,
					max: 1,
					ticks: {
						maxTicksLimit: 4,
						font: {
							size: 9
						}
					},
					grid: {
						color: 'rgba(0,0,0,0.05)'
					}
				}
			},
			plugins: {
				legend: {
					display: false
				},
				tooltip: {
					enabled: true
				}
			}
		}
	});
}

function parseTimestamp(timestampStr) {
	const [datePart, timePart] = timestampStr.split(' ');
	if (!datePart || !timePart) return new Date();
	
	const [year, month, day] = datePart.split('-').map(Number);
	const [hours, minutes, seconds] = timePart.split(':').map(Number);
	
	return new Date(year, month - 1, day, hours, minutes, seconds);
}

let cachedAllData = null;

function fetchSmallChartsData() {
	fetch('/api/noise-data')
	.then(response => response.json())
	.then(data => {
		if (data.length > 0) {
			cachedAllData = data;
			fullTimestamps = data.map(item => item.timestamp);
			
			const allLabels = data.map(item => {
				if (item.timestamp && item.timestamp.includes(' ')) {
					return item.timestamp.split(' ')[1];
				}
				return item.timestamp;
			});
			const allLowFreqData = data.map(item => item.low_freq_ratio);
			const allVolumeData = data.map(item => item.volume);
			
			const latestTimestamp = parseTimestamp(data[data.length - 1].timestamp);
			const cutoffTime = new Date(latestTimestamp.getTime() - selectedTimeRangeMinutes * 60 * 1000);
			
			let startIndex = 0;
			for (let i = data.length - 1; i >= 0; i--) {
				const itemTime = parseTimestamp(data[i].timestamp);
				if (itemTime < cutoffTime) {
					startIndex = i + 1;
					break;
				}
			}
			
			const filteredLabels = allLabels.slice(startIndex);
			const filteredLowFreqData = allLowFreqData.slice(startIndex);
			const filteredVolumeData = allVolumeData.slice(startIndex);
			
			lowFreqChart.data.labels = filteredLabels;
			lowFreqChart.data.datasets[0].data = filteredLowFreqData;
			lowFreqChart.update();
			
			volumeChart.data.labels = filteredLabels;
			volumeChart.data.datasets[0].data = filteredVolumeData;
			volumeChart.update();
		}
	})
	.catch(error => console.error('Error fetching small charts data:', error));
}

function fetchFullChartData() {
	fetch('/api/noise-data')
	.then(response => response.json())
	.then(data => {
		if (data.length > 0) {
			cachedAllData = data;
			const fullLabels = data.map(item => item.timestamp);
			const allLowFreqData = data.map(item => item.low_freq_ratio);
			
			const downsampled = downsampleData(fullLabels, allLowFreqData, MAX_POINTS_FOR_CHART);
			
			fullLowFreqChart.data.labels = downsampled.labels;
			fullLowFreqChart.data.datasets[0].data = downsampled.data;
			fullLowFreqChart.update();
		}
	})
	.catch(error => console.error('Error fetching full chart data:', error));
}

function refreshFullChart() {
	showToast('正在刷新完整历史图表...', 'info');
	fetchFullChartData();
}

function updateEnergyBar(volume) {
	const energyBar = document.getElementById('energy-bar');
	const volumeDisplay = document.getElementById('volume-display');
	
	const maxVolume = 0.05;
	let percentage = Math.min((volume / maxVolume) * 100, 100);
	
	energyBar.style.width = percentage + '%';
	volumeDisplay.textContent = '音量: ' + volume.toFixed(4);
	
	energyBar.classList.remove('glow-green', 'glow-yellow', 'glow-red');
	if (percentage < 33) {
		energyBar.classList.add('glow-green');
	} else if (percentage < 66) {
		energyBar.classList.add('glow-yellow');
	} else {
		energyBar.classList.add('glow-red');
	}
}

function fetchRealtimeData() {
	fetch('/api/noise-data/realtime')
	.then(response => response.json())
	.then(data => {
		document.getElementById('current-low-freq').textContent = data.low_freq_ratio.toFixed(2);
		document.getElementById('current-volume').textContent = data.volume.toFixed(4);
		document.getElementById('current-max-sample').textContent = data.max_sample.toFixed(2);
		
		updateEnergyBar(data.volume);
	})
	.catch(error => console.error('Error fetching realtime data:', error));
}

function fetchData() {
	fetchSmallChartsData();
	fetchRealtimeData();
}

function fetchDetectionLogs() {
	fetch('/api/detection-logs')
	.then(response => response.json())
	.then(logs => {
		const terminalContent = document.getElementById('terminal-content');
		terminalContent.innerHTML = '';
		
		if (logs.length === 0) {
			terminalContent.innerHTML = `
				<div class="terminal-line">
					<span class="terminal-timestamp">[系统]</span>
					<span class="terminal-type-info">等待检测日志...</span>
				</div>
			`;
			return;
		}
		
		logs.forEach(log => {
			const line = document.createElement('div');
			line.className = 'terminal-line';
			
			let typeClass = 'terminal-type-info';
			if (log.type === 'detection') {
				typeClass = 'terminal-type-detection';
			} else if (log.type === 'playback') {
				typeClass = 'terminal-type-playback';
			}
			
			line.innerHTML = `
				<span class="terminal-timestamp">[${log.timestamp}]</span>
				<span class="${typeClass}">${log.message}</span>
			`;
			terminalContent.appendChild(line);
		});
		
		terminalContent.scrollTop = terminalContent.scrollHeight;
	})
	.catch(error => console.error('Error fetching detection logs:', error));
}

initCharts();
fetchData();
fetchFullChartData();
setInterval(fetchData, 1000);
setInterval(fetchFullChartData, 60000);
setInterval(fetchDetectionLogs, 1000);
fetchDetectionLogs();

document.getElementById('refresh-btn').addEventListener('click', fetchData);
document.getElementById('refresh-full-chart-btn').addEventListener('click', refreshFullChart);
document.getElementById('play-random-btn').addEventListener('click', playRandomAudio);
document.getElementById('play-sequence-btn').addEventListener('click', playAudioSequence);
document.getElementById('stop-btn').addEventListener('click', stopAudio);

const timeRangeSelect = document.getElementById('time-range-select');
const customTimeRangeInput = document.getElementById('custom-time-range');

timeRangeSelect.addEventListener('change', function() {
	if (this.value === 'custom') {
		customTimeRangeInput.style.display = 'inline-block';
		customTimeRangeInput.focus();
	} else {
		customTimeRangeInput.style.display = 'none';
		selectedTimeRangeMinutes = parseInt(this.value);
		fetchChartData();
	}
});

customTimeRangeInput.addEventListener('input', function() {
	const value = parseInt(this.value);
	if (value && value > 0) {
		selectedTimeRangeMinutes = value;
		fetchChartData();
	}
});

customTimeRangeInput.addEventListener('keypress', function(e) {
	if (e.key === 'Enter') {
		const value = parseInt(this.value);
		if (value && value > 0) {
			selectedTimeRangeMinutes = value;
			fetchChartData();
		}
	}
});
