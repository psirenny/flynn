import Router from 'marbles/router';
import LandingPageComponent from './views/landing-page';

export default Router.createClass({
	routes: [
		{ path: '', handler: 'landingPage' }
	],

	landingPage: function (params, opts, context) {
		var props = {
			dataStore: context.dataStore
		};
		context.render(LandingPageComponent, props);
	}
});
