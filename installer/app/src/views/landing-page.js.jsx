import AWSLauncher from './aws-launcher';

var LandingPage = React.createClass({
	render: function () {
		return <AWSLauncher dataStore={this.props.dataStore} />;
	}
});
export default LandingPage;
