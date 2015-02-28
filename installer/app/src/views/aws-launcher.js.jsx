import Panel from './panel';
import { green as GreenBtnCSS, disabled as BtnDisabledCSS } from './css/button';
import AWSCredentialsPicker from './aws-credentials-picker';
import AWSRegionPicker from './aws-region-picker';
import AWSInstanceTypePicker from './aws-instance-type-picker';
import IntegerPicker from './integer-picker';
import Dispatcher from '../dispatcher';
import { extend } from 'marbles/utils';

var InstallConfig = React.createClass({
	render: function () {
		return (
			<Panel>
				<form onSubmit={this.__handleSubmit}>
					<AWSCredentialsPicker
						onChange={this.__handleCredentialsChange} />
					<br />
					<br />
					<AWSRegionPicker
						value={this.state.region}
						onChange={this.__handleRegionChange} />
					<br />
					<br />
					<AWSInstanceTypePicker
						value={this.state.instanceType}
						onChange={this.__handleInstanceTypeChange} />
					<br />
					<br />
					<label>
						<div>Number of instances: </div>
						<div style={{
							width: 60
							}}>
							<IntegerPicker
								minValue={1}
								maxValue={5}
								skipValues={[2]}
								value={this.state.numInstances}
								onChange={this.__handleNumInstancesChange} />
						</div>
					</label>
					<br />
					<br />
					<button
						type="submit"
						style={extend({}, GreenBtnCSS,
							this.state.launchBtnDisabled ? BtnDisabledCSS : {})}
						disabled={this.state.launchBtnDisabled}>Launch</button>
					<br />
					<br />
					<div>
						{this.state.installEvents.length > 0 ? (
							this.state.installEvents[this.state.installEvents.length-1].description
						) : null}
					</div>
					{this.state.cert ? (
						<div>
							<h2>Install CA Cert</h2>
							<a href={"data:application/x-x509-ca-cert;base64,"+ this.state.cert}>CA Cert</a>
						</div>
					) : null}
					{this.state.domain && this.state.dashboardLoginToken ? (
						<div>
							Then <a href={"https://dashboard."+ this.state.domain +"?token="+ encodeURIComponent(this.state.dashboardLoginToken)}>Go to Dashboard</a>
						</div>
					) : null}
				</form>
			</Panel>
		);
	},

	getInitialState: function () {
		return this.__getState();
	},

	componentDidMount: function () {
		this.props.dataStore.addChangeListener(this.__handleDataChange);
	},

	__getState: function () {
		var dataStoreState = this.props.dataStore.state;
		return {
			creds: {
				access_key_id: '',
				secret_access_key: ''
			},
			region: 'us-east-1',
			instanceType: 'm3.medium',
			numInstances: 1,
			launchBtnDisabled: dataStoreState.installStarted ? !dataStoreState.installDone : false,

			domain: dataStoreState.domain,
			dashboardLoginToken: dataStoreState.dashboardLoginToken,
			cert: dataStoreState.cert,
			installEvents: dataStoreState.installEvents
		};
	},

	__handleDataChange: function () {
		this.setState(this.__getState());
	},

	__handleCredentialsChange: function (creds) {
		this.setState({
			creds: creds
		});
	},

	__handleRegionChange: function (region) {
		this.setState({
			region: region
		});
	},

	__handleInstanceTypeChange: function (instanceType) {
		this.setState({
			instanceType: instanceType
		});
	},

	__handleNumInstancesChange: function (numInstances) {
		this.setState({
			numInstances: numInstances
		});
	},

	__handleSubmit: function (e) {
		e.preventDefault();
		this.setState({
			launchBtnDisabled: true
		});
		Dispatcher.dispatch({
			name: 'LAUNCH_AWS',
			creds: this.state.creds,
			region: this.state.region,
			instanceType: this.state.instanceType,
			numInstances: this.state.numInstances
		});
	}
});
export default InstallConfig;
