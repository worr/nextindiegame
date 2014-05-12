ng = (function() {
	var template = "A _ about _ in _";

	var getNewGame = function() {
		var xhr = XMLHttpRequest();
		xhr.onreadystatechange = updateName;
		xhr.open("GET", "/api/game/", true);
		xhr.send(null);

		return;

		function updateName() {
			if (xhr.readyState === 4) {
				var text = template;
				var rawData = xhr.responseText;
				var data;

				try {
					data = JSON.parse(rawData);
				} catch (e) {
					document.querySelector("#game").innerHTML = data;
				}

				text = text.replace(/_/, data.Genre).
					replace(/_/, data.Emotion).
					replace(/_/, data.Fantasy);
				document.querySelector("#game").innerHTML = text;
			}
		}
	};

	return {
		getNewGame: getNewGame
	}
})();
