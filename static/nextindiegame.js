ng = (function() {
	var template = "A _ about _ in _";

	var getNewGame = function() {
		if (window.location.pathname.match(/\/l\//)) {
			window.location = "/";
			return;
		}

		var xhr = new XMLHttpRequest();
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

				if (data.Genre) {
					text = text.replace(/_/, data.Genre).
						replace(/_/, data.Emotion).
						replace(/_/, data.Fantasy);
					if (data.Genre.match(/^[AEIOUaeiou](?!ne\b)/)) {
						text = text.replace(/^A/, "An");
					}
				} else {
					text = data.Error
				}

				document.querySelector("#game").innerHTML = text;

				var link = document.querySelector("#permalink");
				link.href = data.Link;
			}
		}
	};

	return {
		getNewGame: getNewGame
	}
})();
