var AWSCredentialsPicker = React.createClass({
	getDefaultProps: function () {
		return {
			inputCSS: {
				width: 280
			}
		};
	},

	render: function () {
		return (
			<label>
				<div>AWS Credentials: </div>
				<div style={{
						marginLeft: 10
					}}>
					<br />
					<label>
						<div>Access Key ID: </div>
						<input
							ref="key"
							type="text"
							style={this.props.inputCSS}
							placeholder="AWS_ACCESS_KEY_ID"
							onChange={this.__handleChange} />
					</label>
					<br />
					<br />
					<label>
						<div>Secret Access Key: </div>
						<input
							ref="secret"
							type="text"
							style={this.props.inputCSS}
							placeholder="AWS_SECRET_ACCESS_KEY"
							onChange={this.__handleChange} />
					</label>
				</div>
				<p>Defaults to using the appropriate env vars</p>
			</label>
		);
	},

	__handleChange: function () {
		this.props.onChange({
			access_key_id: this.refs.key.getDOMNode().value.trim(),
			secret_access_key: this.refs.secret.getDOMNode().value.trim()
		});
	}
});
export default AWSCredentialsPicker;
