package rundeck

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

type JobSummary struct {
	XMLName     xml.Name `xml:"job"`
	ID          string   `xml:"id,attr"`
	Name        string   `xml:"name"`
	GroupName   string   `xml:"group"`
	ProjectName string   `xml:"project"`
	Description string   `xml:"description,omitempty"`
}

type jobSummaryList struct {
	XMLName xml.Name     `xml:"jobs"`
	Jobs    []JobSummary `xml:"job"`
}

type JobDetail struct {
	XMLName                   xml.Name           `xml:"job"`
	ID                        string             `xml:"id,omitempty"`
	Name                      string             `xml:"name"`
	GroupName                 string             `xml:"group,omitempty"`
	ProjectName               string             `xml:"context>project,omitempty"`
	Description               string             `xml:"description,omitempty"`
	LogLevel                  string             `xml:"loglevel",omitempty`
	AllowConcurrentExecutions bool               `xml:"multipleExecutions"`
	OptionsConfig             JobOptions         `xml:"context>options"`
	MaxThreadCount            int                `xml:"dispatch>threadcount"`
	ContinueOnError           bool               `xml:"dispatch>keepgoing"`
	RankAttribute             string             `xml:"dispatch>rankAttribute"`
	RankOrder                 string             `xml:"dispatch>rankOrder"`
	CommandSequence           JobCommandSequence `xml:"sequence"`
	NodeFilter                JobNodeFilter      `xml:"nodefilters"`
}

type jobDetailList struct {
	XMLName xml.Name    `xml:"joblist"`
	Jobs    []JobDetail `xml:"job"`
}

type JobOptions struct {
	PreserveOrder bool        `xml:"preserveOrder"`
	Options       []JobOption `xml:"option"`
}

type JobOption struct {
	XMLName                 xml.Name `xml:"option"`
	DefaultValue            string   `xml:"value,attr,omitempty"`
	ValueChoices            []string `xml:"values,attr"`
	ValueChoicesURL         string   `xml:"valuesUrl,attr,omitempty"`
	RequirePredefinedChoice bool     `xml:"enforcedvalues,attr"`
	ValidationRegex         string   `xml:"regex,attr,omitempty"`
	Description             string   `xml:"description,omitempty"`
	IsRequired              bool     `xml:"required,attr"`
	AllowsMultipleValues    bool     `xml:"multivalued,attr"`
	MultiValueDelimiter     string   `xml:"delimeter,attr"`
	ObscureInput            bool     `xml:"secure,attr"`
	ValueIsExposedToScripts bool     `xml:"valueExposed,attr"`
}

type JobValueChoices []string

type JobCommandSequence struct {
	XMLName          xml.Name     `xml:"sequence"`
	ContinueOnError  bool         `xml:"keepgoing,attr"`
	OrderingStrategy string       `xml:"strategy,attr"`
	Commands         []JobCommand `xml:"command"`
}

type JobCommand struct {
	XMLName        xml.Name
	ShellCommand   string            `xml:"exec"`
	Script         string            `xml:"script"`
	ScriptFile     string            `xml:"scriptfile"`
	ScriptFileArgs string            `xml:"scriptargs"`
	Job            *JobCommandJobRef `xml:"jobref"`
	StepPlugin     *JobPlugin        `xml:"step-plugin"`
	NodeStepPlugin *JobPlugin        `xml:"node-step-plugin"`
}

type JobCommandJobRef struct {
	XMLName        xml.Name                  `xml:"jobref"`
	Name           string                    `xml:"name,attr"`
	GroupName      string                    `xml:"group,attr"`
	RunForEachNode bool                      `xml:"nodeStep,attr"`
	Arguments      JobCommandJobRefArguments `xml:"arg"`
}

type JobCommandJobRefArguments string

type JobPlugin struct {
	XMLName xml.Name
	Type    string          `xml:"type,attr"`
	Config  JobPluginConfig `xml:"configuration"`
}

type JobPluginConfig map[string]string

type JobNodeFilter struct {
	ExcludePrecedence bool   `xml:"excludeprecedence"`
	Query             string `xml:"filter"`
}

func (c *Client) GetJobsForProject(projectName string) ([]JobSummary, error) {
	jobList := &jobSummaryList{}
	err := c.get([]string{"project", projectName, "jobs"}, nil, jobList)
	return jobList.Jobs, err
}

func (c *Client) GetJob(uuid string) (*JobDetail, error) {
	jobList := &jobDetailList{}
	err := c.get([]string{"job", uuid}, nil, jobList)
	if err != nil {
		return nil, err
	}
	return &jobList.Jobs[0], nil
}

func (c JobValueChoices) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(c) > 0 {
		return xml.Attr{name, strings.Join(c, ",")}, nil
	} else {
		return xml.Attr{}, nil
	}
}

func (c *JobValueChoices) UnmarshalXMLAttr(attr xml.Attr) error {
	values := strings.Split(attr.Value, ",")
	*c = values
	return nil
}

func (a JobCommandJobRefArguments) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		xml.Attr{xml.Name{Local: "line"}, string(a)},
	}
	e.EncodeToken(start)
	e.EncodeToken(xml.EndElement{start.Name})
	return nil
}

func (a *JobCommandJobRefArguments) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type jobRefArgs struct {
		Line string `xml:"line,attr"`
	}
	args := jobRefArgs{}
	d.DecodeElement(&args, &start)

	*a = JobCommandJobRefArguments(args.Line)

	return nil
}

func (c JobPluginConfig) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(map[string]string(c)) == 0 {
		return nil
	}
	e.EncodeToken(start)

	// Sort the keys so we'll have a deterministic result.
	keys := []string{}
	for k, _ := range map[string]string(c) {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := c[k]
		e.EncodeToken(xml.StartElement{
			Name: xml.Name{Local: "entry"},
			Attr: []xml.Attr{
				xml.Attr{
					Name:  xml.Name{Local: "key"},
					Value: k,
				},
				xml.Attr{
					Name:  xml.Name{Local: "value"},
					Value: v,
				},
			},
		})
		e.EncodeToken(xml.EndElement{xml.Name{Local: "entry"}})
	}
	e.EncodeToken(xml.EndElement{start.Name})
	return nil
}

func (c *JobPluginConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	result := map[string]string{}
	for {
		token, err := d.Token()
		if token == nil {
			err = fmt.Errorf("EOF while decoding job command plugin config")
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		default:
			return fmt.Errorf("unexpected token %t while decoding job command plugin config", t)
		case xml.StartElement:
			if t.Name.Local != "entry" {
				return fmt.Errorf("unexpected element %s while looking for plugin config entries", t.Name.Local)
			}
			var k string
			var v string
			for _, attr := range t.Attr {
				if attr.Name.Local == "key" {
					k = attr.Value
				} else if attr.Name.Local == "value" {
					v = attr.Value
				}
			}
			if k == "" {
				return fmt.Errorf("found plugin config entry with empty key")
			}
			result[k] = v
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				*c = result
				return nil
			}
		}
	}
}