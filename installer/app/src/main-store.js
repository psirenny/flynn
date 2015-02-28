import { createClass } from 'marbles/utils';
import State from 'marbles/state';
import Client from './client';

export default createClass({
	mixins: [State],

	registerWithDispatcher: function (dispatcher) {
		this.dispatcherIndex = dispatcher.register(this.handleEvent.bind(this));
	},

	willInitialize: function () {
		this.state = this.getInitialState();
		this.__changeListeners = [];
	},

	getInitialState: function () {
		return {
			installEvents: [],
			installID: null,
			installStarted: false,
			installDone: false,
			domain: null,
			dashboardLoginToken: null,
			cert: null
		};
	},

	handleEvent: function (event) {
		switch (event.name) {
			case 'LAUNCH_AWS':
				this.launchAWS(event.creds, event.region, event.instanceType, event.numInstances);
			break;

			case 'LAUNCH_INSTALL_SUCCESS':
				this.setState({
					installID: event.res.id
				});
				Client.openEventStream(event.res.id);
			break;

			case 'LAUNCH_INSTALL_FAILURE':
				window.console.error(event);
			break;

			case 'INSTALL_PROMPT_REQUESTED':
				if (event.data.type === 'yes_no') {
					Client.sendPromptResponse(this.state.installID, {
						yes: window.confirm(event.data.message)
					});
				} else {
					Client.sendPromptResponse(this.state.installID, {
						input: window.prompt(event.data.message)
					});
				}
			break;

			case 'DOMAIN':
				this.setState({
					domain: event.domain
				});
			break;

			case 'DASHBOARD_LOGIN_TOKEN':
				this.setState({
					dashboardLoginToken: event.token
				});
			break;

			case 'CA_CERT':
				this.setState({
					cert: event.cert
				});
			break;

			case 'INSTALL_EVENT':
				this.handleInstallEvent(event.data);
			break;

			case 'INSTALL_DONE':
				this.setState({
					installDone: true,
					installStarted: false
				});
			break;
		}
	},

	handleInstallEvent: function (data) {
		this.setState({
			installEvents: this.state.installEvents.concat([data])
		});
	},

	launchAWS: function (creds, region, instanceType, numInstances) {
		this.setState({
			installEvents: [],
			installID: null,
			installStarted: true,
			installDone: false,
			domain: null,
			dashboardLoginToken: null,
			cert: null
		});
		Client.launchInstall({
			creds: creds,
			region: region,
			instance_type: instanceType,
			num_instances: numInstances
		});
	}
});
