import React from "react";
import map from "lodash/map";
import after from "lodash/after";
import forEach from "lodash/forEach";

export default class FileInput extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      errText: "",
      fileAdded: false
    }
  }

  handleRemoveFile = (name, item) => {
    if (!item) {
      // single file remove
      this.props.onChange([{ filename: "", value: ""}]);
      this.setState({ fileAdded: false });
    } else {
      // variadic config item remove
      this.props.handleRemoveFile(name, item)
    }
  }

  handleOnChange = (ev) => {
    this.setState({ errText: "" });

    let files = [];
    let error;

    const done = after(ev.target.files.length, () => {
      // this.refs.file.getDOMNode().value = "";
      if (error) {
        this.setState({ errText: error });
      } else if (this.props.onChange) {
        this.setState({ fileAdded: true })
        this.props.onChange(files);
      }
    });

    forEach(ev.target.files, (file) => {
      var reader = new FileReader();
      reader.onload = () => {
        var vals = reader.result.split(",");
        if (vals.length !== 2) {
          error = "Invalid file data";
        } else {
          files.push({ value: file.name, filename: vals[1] });
        }
        done();
      };
      reader.readAsDataURL(file);
    });
  }

  renderFilesUploaded = (arr) => {
    if (!arr || arr.length === 0) { return null };
    return arr.map((item, index) => {
      return (
        <div key={`${item}-${index}`} className="u-marginTop--10">
          <span className={`icon u-smallCheckGreen u-marginRight--10 u-top--3`}></span>
          {item}
          {arr.length > 1 ? <span onClick={() => this.handleRemoveFile(this.props.name, item)} className="icon gray-trash-small clickable u-marginLeft--5 u-top--3" /> : null}
        </div>
      );
    });
  }

  render() {
    let label;
    this.props.label ? label = this.props.label : this.props.multiple
      ? label = "Upload files" : label = "Upload a file";
    const hasFileOrValue = this.state.fileAdded || this.props.value || (this.props.multiple && this.props.filenamesText !== "");

    return (
      <div>
        <div className={`${this.props.readonly ? "readonly" : ""} ${this.props.disabled ? "disabled" : ""}`}>
          <p className="sub-header-color field-section-sub-header u-marginTop--10 u-marginBottom--5">{label}</p>
          <div className="flex flex-row">
            <div className={`${hasFileOrValue ? "file-uploaded" : "custom-file-upload"}`}>
              <input
                ref={(file) => this.file = file}
                type="file"
                name={this.props.name}
                className="inputfile"
                id={`${this.props.name} selector`}
                onChange={this.handleOnChange}
                readOnly={this.props.readOnly}
                multiple={this.props.multiple}
                disabled={this.props.disabled}
              />
              {!this.props.multiple ?
                    hasFileOrValue ? 
                      <div>
                        <div>
                          <span className={`icon u-smallCheckGreen u-marginRight--10 u-top--3`}></span>
                          {this.props.filenamesText}
                          <span onClick={() => this.handleRemoveFile(this.props.name)} className="icon gray-trash-small clickable u-marginLeft--5 u-top--3" />
                        </div>
                        <p className="u-linkColor u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--5">Select a different file</p>
                      </div>
                    :
                      <label htmlFor={`${this.props.name} selector`} className="u-position--relative">
                        <span className={`icon u-ovalIcon clickable u-marginRight--10 u-top--3`}></span>
                        Browse files for {this.props.title}
                      </label>
              :
                <div>
                  {this.renderFilesUploaded(this.props.filenamesText)}
                  <label htmlFor={`${this.props.name} selector`} className="u-position--relative">
                    {hasFileOrValue ? 
                      <p className="u-linkColor u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--10">Select other files</p>
                    : `Browse files for ${this.props.title}` }
                  </label>
                </div>
              }
            </div>
          </div>
        </div>
        <small className="text-danger"> {this.state.errText}</small>
      </div>
    );
  }
}
