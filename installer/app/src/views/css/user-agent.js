export default {
	isFirefox: function () {
		return window.navigator.userAgent.match(/\bfirefox\b/i) !== null;
	},

	isSafari: function () {
		return window.navigator.userAgent.match(/\bsafari\b/i) !== null;
	}
};
