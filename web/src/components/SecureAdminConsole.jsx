import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { Utilities } from "../utilities/utilities";
import { loginToAdminConsole } from "../mutations/AuthMutations";
import "../scss/components/Login.scss";

class SecureAdminConsole extends React.Component {

  state = {
    password: "",
    passwordErr: false,
    passwordErrMessage: "",
    authLoading: false,
  }

  completeLogin = (data) => {
    const { onLoginSuccess } = this.props;
    let token = data.loginToAdminConsole.token
    if (Utilities.localStorageEnabled()) {
      window.localStorage.setItem("token", token);
      onLoginSuccess().then((res) => {
        this.setState({ authLoading: false });
        if (res.length > 0) {
          this.props.history.replace(`/app/${res[0].slug}`);
        } else {
          this.props.history.replace("upload-license");
        }
      });
    } else {
      this.props.history.push("/unsupported");
    }
  }

  validatePassword = () => {
    const { password } = this.state;
    if (!password || password.length === "0") {
      this.setState({
        passwordErr: true,
        passwordErrMessage: `Please provide your password`,
      });
      return false;
    }
    return true;
  }

  loginToConsole = async () => {
    const { password } = this.state;
    if (this.validatePassword()) {
      this.setState({ authLoading: true });
      try {
        const res = await this.props.loginToAdminConsole(password);
        this.completeLogin(res.data);
      } catch (error) {
        error.graphQLErrors.map(({ message }) => {
          let json;

          try {
            json = JSON.parse(message);
            this.setState({
              authLoading: false,
              passwordErr: true,
              passwordErrMessage: json.replicatedMessage || "There was an error logging in. Please try again",
            });
          } catch (error) {
            this.setState({
              authLoading: false,
              passwordErr: true,
              passwordErrMessage: message,
            });
          }
        });
      }
    }
  }

  submitForm = (e) => {
    const enterKey = e.keyCode === 13;
    if (enterKey) {
      e.preventDefault();
      e.stopPropagation();
      this.loginToConsole();
    }
  }

  componentDidMount() {
    window.addEventListener("keydown", this.submitForm);
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.submitForm);
  }

  render() {
    const {
      appName,
      logo,
    } = this.props;
    const {
      password,
      authLoading,
      passwordErr,
      passwordErrMessage,
    } = this.state;

    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              {logo ?
                <span className="icon brand-login-icon" style={{ backgroundImage: `url(${logo})` }} />
              :
                <span className="icon ship-login-icon" />
              }
              <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-lineHeight--more u-color--tuna u-fontWeight--bold">Log in to {appName}</p>
            </div>
            <p className="u-marginTop--10 u-marginTop--5 u-fontSize--large u-textAlign--center u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
              Enter the password to access the {appName} admin console.
            </p>
            <div className="u-marginTop--20 flex-column">
              {passwordErr && <p className="u-fontSize--normal u-fontWeight--medium u-color--chestnut u-lineHeight--normal u-marginBottom--20">{passwordErrMessage}</p>}
              <div>
                <div className="component-wrapper">
                  <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password} onChange={(e) => { this.setState({ password: e.target.value }) }}/>
                </div>
                <div className="u-marginTop--20 flex">
                  <button type="submit" className="btn primary" disabled={authLoading} onClick={this.loginToConsole}>{authLoading ? "Logging in" : "Log in"}</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(loginToAdminConsole, {
    props: ({ mutate }) => ({
      loginToAdminConsole: (password) => mutate({ variables: { password } })
    })
  }),
)(SecureAdminConsole);
