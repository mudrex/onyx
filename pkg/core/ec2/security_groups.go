package ec2

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	ec2Lib "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mudrex/onyx/pkg/core/iam"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
)

var reverseAllowedRules = map[int32]string{
	22:    "ssh",
	6379:  "redis",
	27017: "mongo",
	3306:  "mysql",
	5432:  "timescale",
	6432:  "pgbouncer",
}

var allowedRules = map[string]int32{
	"ssh":       22,
	"redis":     6379,
	"mongo":     27017,
	"mysql":     3306,
	"timescale": 5432,
	"pgbouncer": 6432,
}

type SecurityGroup struct {
	ID           string
	Name         string
	Description  string
	Tags         []types.Tag
	rules        []SecurityGroupRule
	allowedRules map[string]bool // extracted from tag: "onyx:rules"
}

type SecurityGroupRule struct {
	user        string
	port        int32
	cidr        string
	protocol    string
	description string
}

type Filter struct {
	Key   string
	Value string
}

type SecurityGroupToAlter struct {
	SecurityGroup SecurityGroup
	Ports         map[int32]bool
}

func ExtractFilter(filterStr string) *Filter {
	filterArray := strings.Split(filterStr, "=")
	if len(filterArray) == 2 {
		return &Filter{
			Key:   strings.ToLower(filterArray[0]),
			Value: strings.ToLower(filterArray[1]),
		}
	}

	return &Filter{}
}

func (sg *SecurityGroup) Refresh(ctx context.Context, cfg aws.Config) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		GroupIds: []string{sg.ID},
	})
	if err != nil {
		logger.Error("Unable to refresh security group")
		return
	}

	securityGroups := convertSecurityGroups(&output.SecurityGroups)

	sg.rules = securityGroups[0].rules
}

func NewSecurityGroupByName(ctx context.Context, cfg aws.Config, name string) (*SecurityGroup, error) {
	securityGroup := SecurityGroup{
		rules: make([]SecurityGroupRule, 0),
	}

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
		},
	})
	if err != nil {
		return &securityGroup, err
	}

	if len(output.SecurityGroups) == 1 {
		securityGroup.ID = aws.ToString((output.SecurityGroups[0].GroupId))
		securityGroup.Name = aws.ToString((output.SecurityGroups[0].GroupName))
		securityGroup.Description = aws.ToString((output.SecurityGroups[0].Description))
	}

	return &securityGroup, nil
}

func NewSecurityGroup(ctx context.Context, cfg aws.Config, id string) (*SecurityGroup, error) {
	securityGroup := SecurityGroup{
		ID:    id,
		rules: make([]SecurityGroupRule, 0),
	}

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		GroupIds: []string{id},
	})
	if err != nil {
		return &securityGroup, err
	}

	securityGroups := convertSecurityGroups(&output.SecurityGroups)

	return &(securityGroups[0]), nil
}

func NewSecurityGroupRule(port int32, user string) (*SecurityGroupRule, error) {
	return &SecurityGroupRule{
		port: port,
		user: strings.ToLower(user),
	}, nil
}

// DisplaySecurityGroup prints the security group details
func (sg *SecurityGroup) DisplaySecurityGroup(ctx context.Context, cfg aws.Config, changedRules *[]SecurityGroupRule, toRemove bool) {
	changedRulesMap := make(map[string]bool)
	if changedRules != nil {
		for _, rule := range *changedRules {
			changedRulesMap[fmt.Sprintf("%s_%d", rule.description, rule.port)] = true
		}
	}

	if toRemove {
		logger.Warn("Proposed changes to security group: " + logger.Bold(sg.ID))
	} else {
		logger.Info("Current state of security group: " + logger.Bold(sg.ID))
	}

	fmt.Println("|-----------------------------------------------------")
	fmt.Println(fmt.Sprintf("| %s (%s)", logger.Bold(sg.Name), sg.ID))
	fmt.Println("| Description:", sg.Description)
	fmt.Println("| Rules:")
	fmt.Println("|  |------------------------------------------")
	for _, rule := range sg.rules {
		if _, ok := changedRulesMap[fmt.Sprintf("%s_%d", rule.description, rule.port)]; ok {
			if toRemove {
				fmt.Print(logger.Red(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", rule.port, rule.port, rule.protocol, rule.cidr, rule.description)))
				fmt.Println(logger.Bold("    <------- This rule will be removed/updated"))
			} else {
				fmt.Println(logger.Green(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", rule.port, rule.port, rule.protocol, rule.cidr, rule.description)))
			}
		} else {
			fmt.Println(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", rule.port, rule.port, rule.protocol, rule.cidr, rule.description))
		}

		// for _, a := range ipPermission.UserIdGroupPairs {
		// 	fmt.Println(fmt.Sprintf("|  | %d -> %d (%s): %s - %v", ipPermission.FromPort, ipPermission.ToPort, *ipPermission.IpProtocol, *a.GroupId, aws.ToString(a.Description)))
		// }
	}
	fmt.Println("|  |------------------------------------------")
	fmt.Println("|-----------------------------------------------------")
}

func (sgRule *SecurityGroupRule) enrichRuleDescription() string {
	return fmt.Sprintf("[Onyx approved] User: %s", sgRule.user)
}

func (sgRule *SecurityGroupRule) GetUserName() string {
	return strings.Replace(sgRule.description, "[Onyx approved] User: ", "", -1)
}

func (sgRule *SecurityGroupRule) attachNewIP(ip string) {
	sgRule.cidr = ip
}

func (sg *SecurityGroup) FilterIngressRules(securityGroupRules *[]SecurityGroupRule) (filteredSecurityGroupRules []SecurityGroupRule) {
	for _, sgRule := range sg.rules {
		for _, securityGroupRule := range *securityGroupRules {
			if sgRule.port == securityGroupRule.port && (strings.Contains(sgRule.description, securityGroupRule.user) || sgRule.description == securityGroupRule.enrichRuleDescription()) {
				filteredSecurityGroupRules = append(filteredSecurityGroupRules, sgRule)
			}
		}
	}
	return
}

func (sg *SecurityGroup) Authorize(ctx context.Context, cfg aws.Config, rules *[]SecurityGroupRule, publicIP string) {
	for _, existingRule := range sg.rules {
		if existingRule.cidr == publicIP {
			logger.Success("CIDR is already whitelisted by %s", logger.Bold(existingRule.GetUserName()))
			return
		}
	}

	logger.Info("Authorizing new rules for %s", logger.Bold(sg.ID))
	ec2Handler := ec2Lib.NewFromConfig(cfg)

	ipPermissions := make([]types.IpPermission, 0)
	for i, rule := range *rules {
		if _, ok := sg.allowedRules[reverseAllowedRules[rule.port]]; !ok {
			logger.Error("Not processing ports %s for %s", logger.Underline(rule.port), logger.Bold(sg.ID))
			continue
		}

		// Update rule ip and description
		rule.description = rule.enrichRuleDescription()
		rule.attachNewIP(publicIP)

		ipPermissions = append(ipPermissions, types.IpPermission{
			FromPort:   rule.port,
			IpProtocol: aws.String("tcp"),
			ToPort:     rule.port,
			IpRanges: []types.IpRange{
				{
					CidrIp:      aws.String(rule.cidr),
					Description: aws.String(rule.description),
				},
			},
		})

		(*rules)[i] = rule
	}

	if len(ipPermissions) > 0 {
		_, err := ec2Handler.AuthorizeSecurityGroupIngress(ctx, &ec2Lib.AuthorizeSecurityGroupIngressInput{
			GroupId:       aws.String(sg.ID),
			IpPermissions: ipPermissions,
		})
		if err != nil {
			panic(err)
		}

		logger.Success("Authorized new rules for %s", logger.Bold(sg.ID))
	} else {
		logger.Success("Nothing to do %s", logger.Bold(sg.ID))
	}

	// Force refresh
	sg, _ = NewSecurityGroup(ctx, cfg, sg.ID)
	sg.DisplaySecurityGroup(ctx, cfg, rules, false)
}

// Revoke revokes the given rules from security groups
func (sg *SecurityGroup) Revoke(ctx context.Context, cfg aws.Config, rules []SecurityGroupRule, newCidr string, followedByAuthorize bool) error {
	if len(rules) == 0 {
		return nil
	}

	ipranges := make([]types.IpRange, 0)
	for _, rule := range rules {
		ipranges = append(ipranges, types.IpRange{
			CidrIp:      aws.String(rule.cidr),
			Description: aws.String(rule.description),
		})
	}

	// if len(ipranges) == 0 {
	// 	logger.Warn("No changes to be done for %s. IP(%s) didn't change", sg.Name, newCidr)
	// 	return nil
	// }

	sg.DisplaySecurityGroup(ctx, cfg, &rules, true)

	logger.Info("Revoking old rules for %s", logger.Bold(sg.ID))

	ipPermissions := []types.IpPermission{
		{
			FromPort:   rules[0].port,
			ToPort:     rules[0].port,
			IpProtocol: aws.String("tcp"),
			IpRanges:   ipranges,
		},
	}

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.RevokeSecurityGroupIngress(ctx, &ec2Lib.RevokeSecurityGroupIngressInput{
		GroupId:       aws.String(sg.ID),
		IpPermissions: ipPermissions,
	})
	if err != nil {
		return err
	}

	// Force refresh
	sg.Refresh(ctx, cfg)

	if output.Return {
		logger.Success("Revoked old rules for %s", logger.Bold(sg.ID))
		sg.DisplaySecurityGroup(ctx, cfg, nil, false)
		return nil
	}

	sg.DisplaySecurityGroup(ctx, cfg, nil, false)
	return errors.New("unable to revoke old rules")
}

func SelectSecurityGroups(
	ctx context.Context,
	cfg aws.Config,
	env string,
	filters *[]Filter,
	skipChoice bool,
	baseRuleType []int32,
) (map[string]SecurityGroupToAlter, error) {
	ruleTypeSecurityGroupsMap := make(map[string]SecurityGroupToAlter)

	allSecurityGroups, err := ListSecurityGroupsByEnv(ctx, cfg, env)
	if err != nil {
		return ruleTypeSecurityGroupsMap, err
	}

	if len(allSecurityGroups) == 0 {
		return ruleTypeSecurityGroupsMap, nil
	}

	securityGroups := applyFilters(&allSecurityGroups, filters)
	if len(*securityGroups) == 0 {
		return ruleTypeSecurityGroupsMap, nil
	}

	if len(*securityGroups) == 1 && skipChoice {
		securityGroupToAlter := SecurityGroupToAlter{
			SecurityGroup: (*securityGroups)[0],
			Ports:         make(map[int32]bool),
		}

		for _, port := range baseRuleType {
			securityGroupToAlter.Ports[port] = true
		}

		if len(securityGroupToAlter.Ports) == 0 {
			return ruleTypeSecurityGroupsMap, errors.New("no rules to authorize")
		}

		ruleTypeSecurityGroupsMap[(*securityGroups)[0].ID] = securityGroupToAlter

		return ruleTypeSecurityGroupsMap, nil
	}

	logger.Info("Select security groups:")
	for i, securityGroup := range *securityGroups {
		fmt.Println(logger.Bold(i), ":", securityGroup.ID, "(", logger.Italic(securityGroup.Name), ")")
	}

	choices := utils.GetUserInput("Enter Choice: ")

	if len(choices) == 0 {
		return nil, errors.New(logger.Bold("Invalid choice"))
	}

	if strings.Contains(choices, ":") {
		for _, choice := range strings.Split(choices, " ") {
			choiceArr := strings.Split(choice, ":")
			if len(choiceArr) != 2 {
				continue
			}

			index := choiceArr[0]
			ruleTypeArr := strings.Split(strings.TrimSpace(choiceArr[1]), ",")

			i, _ := strconv.ParseInt(strings.TrimSpace(index), 0, 32)
			if i >= 0 && i < int64(len(*securityGroups)) {
				securityGroup := (*securityGroups)[int(i)]

				securityGroupToAlter := SecurityGroupToAlter{
					SecurityGroup: securityGroup,
					Ports:         make(map[int32]bool),
				}

				if value, ok := ruleTypeSecurityGroupsMap[securityGroup.ID]; ok {
					securityGroupToAlter = value
				}

				for _, rule := range ruleTypeArr {
					if value, ok := allowedRules[strings.ToLower(rule)]; ok {
						securityGroupToAlter.Ports[value] = true
					}
				}

				ruleTypeSecurityGroupsMap[securityGroup.ID] = securityGroupToAlter
			}
		}
	} else {
		for _, index := range strings.Split(choices, ",") {
			i, _ := strconv.ParseInt(strings.TrimSpace(index), 0, 32)
			if i >= 0 && i < int64(len(*securityGroups)) {
				securityGroupToAlter := SecurityGroupToAlter{
					SecurityGroup: (*securityGroups)[i],
					Ports:         make(map[int32]bool),
				}

				for _, port := range baseRuleType {
					securityGroupToAlter.Ports[port] = true
				}

				if len(securityGroupToAlter.Ports) == 0 {
					continue
				}

				ruleTypeSecurityGroupsMap[(*securityGroups)[i].ID] = securityGroupToAlter
			}
		}
	}

	if len(ruleTypeSecurityGroupsMap) == 0 {
		return ruleTypeSecurityGroupsMap, errors.New("no rules to authorize")
	}

	return ruleTypeSecurityGroupsMap, nil
}

// ListSecurityGroupsByEnv returns all security groups filtered by Tag:Environment if provided
func ListSecurityGroupsByEnv(ctx context.Context, cfg aws.Config, env string) ([]SecurityGroup, error) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)
	filters := make([]types.Filter, 0)

	if env != "" {
		filters = append(filters, types.Filter{
			Name:   aws.String("tag:Environment"),
			Values: []string{strings.Title(strings.ToLower(env))},
		})
	} else {
		logger.Warn("Please use `--env` to narrow down search.")
	}

	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return convertSecurityGroups(&output.SecurityGroups), nil
}

func convertSecurityGroups(libSecurityGroups *[]types.SecurityGroup) (securityGroups []SecurityGroup) {
	for _, securityGroup := range *libSecurityGroups {
		rules := make([]SecurityGroupRule, 0)
		for _, ipp := range securityGroup.IpPermissions {
			for _, iprange := range ipp.IpRanges {
				rules = append(rules, SecurityGroupRule{
					cidr:        *iprange.CidrIp,
					description: aws.ToString(iprange.Description),
					port:        ipp.FromPort,
					protocol:    aws.ToString(ipp.IpProtocol),
				})
			}

			for _, group := range ipp.UserIdGroupPairs {
				rules = append(rules, SecurityGroupRule{
					cidr:        *group.GroupId,
					description: aws.ToString(group.Description),
					port:        ipp.FromPort,
					protocol:    aws.ToString(ipp.IpProtocol),
				})
			}
		}

		allowedRulesForSecurityGroup := make(map[string]bool)
		isBlocked := true
		for _, tag := range securityGroup.Tags {
			if aws.ToString(tag.Key) == "onyx:rules" {
				for _, rt := range strings.Split(aws.ToString(tag.Value), ",") {
					allowedRulesForSecurityGroup[rt] = true
					isBlocked = false
					break
				}
			}
		}

		if !isBlocked {
			securityGroups = append(securityGroups, SecurityGroup{
				ID:           *securityGroup.GroupId,
				Name:         aws.ToString(securityGroup.GroupName),
				Tags:         securityGroup.Tags,
				Description:  aws.ToString(securityGroup.Description),
				rules:        rules,
				allowedRules: allowedRulesForSecurityGroup,
			})
		}
	}

	return
}

func applyFilters(securityGroups *[]SecurityGroup, filters *[]Filter) *[]SecurityGroup {
	if filters == nil || len(*filters) == 0 {
		return securityGroups
	}

	filteredSecurityGroups := make([]SecurityGroup, 0)

	for _, filter := range *filters {
		if filter.Key == "name" {
			for _, securityGroup := range *securityGroups {
				if strings.Contains(securityGroup.Name, filter.Value) {
					filteredSecurityGroups = append(filteredSecurityGroups, securityGroup)
				}
			}
		}

		if filter.Key == "type" {
			for _, securityGroup := range *securityGroups {
				if _, ok := securityGroup.allowedRules[filter.Value]; ok {
					filteredSecurityGroups = append(filteredSecurityGroups, securityGroup)
				}
			}
		}
	}

	return &filteredSecurityGroups
}

func AuthorizeOrRevokeRule(
	envOrIDOrName string,
	types []string,
	ports []int32,
	filters []string,
	skipChoice,
	authorize bool,
) error {
	filtersToApply := make([]Filter, 0)
	if len(filters) > 0 {
		for _, filter := range filters {
			filterObj := ExtractFilter(filter)
			if filterObj.Key != "" && filterObj.Value != "" {
				filtersToApply = append(filtersToApply, *filterObj)
			}
		}
	}

	for _, t := range types {
		filtersToApply = append(filtersToApply, Filter{
			Key:   "type",
			Value: t,
		})
	}

	portsToUpdate := make([]int32, 0)
	for _, t := range types {
		if value, ok := allowedRules[strings.ToLower(t)]; ok {
			portsToUpdate = append(portsToUpdate, value)
		} else {
			allowedRulesArray := make([]string, len(allowedRules))

			i := 0
			for r := range allowedRules {
				allowedRulesArray[i] = r
				i++
			}
			return errors.New("invalid type. Allowed values: " + strings.Join(allowedRulesArray, "|"))
		}
	}

	portsToUpdate = append(portsToUpdate, ports...)

	if len(portsToUpdate) == 0 {
		return errors.New("no ports to authorize")
	}

	securityGroupUser, err := iam.Whoami()
	if err != nil {
		return errors.New("Unable to derive username. Error: " + err.Error())
	}

	if securityGroupUser == "" || len(securityGroupUser) < 3 {
		return errors.New("invalid user")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ctx := context.Background()

	securityGroups := make(map[string]SecurityGroupToAlter)
	if strings.HasPrefix(envOrIDOrName, "sg-") {
		securityGroup, err := NewSecurityGroup(ctx, cfg, envOrIDOrName)
		if err != nil {
			return errors.New("Invalid security group id. Error: " + err.Error())
		}

		logger.Success("Detected security group: %s (%s)", logger.Bold(securityGroup.ID), logger.Italic(securityGroup.Name))

		securityGroups[securityGroup.ID] = SecurityGroupToAlter{
			SecurityGroup: *securityGroup,
			Ports:         make(map[int32]bool),
		}

		for _, port := range portsToUpdate {
			if _, ok := securityGroups[securityGroup.ID].Ports[port]; !ok {
				securityGroups[securityGroup.ID].Ports[port] = true
			}
		}
	} else {
		securityGroup, err := NewSecurityGroupByName(ctx, cfg, envOrIDOrName)
		if err != nil {
			return errors.New("Invalid security group name. Error: " + err.Error())
		}

		if securityGroup.ID != "" {
			logger.Success("Detected security group: %s (%s)", logger.Bold(securityGroup.ID), logger.Italic(securityGroup.Name))

			securityGroups[securityGroup.ID] = SecurityGroupToAlter{
				SecurityGroup: *securityGroup,
				Ports:         make(map[int32]bool),
			}

			for _, port := range portsToUpdate {
				if _, ok := securityGroups[securityGroup.ID].Ports[port]; !ok {
					securityGroups[securityGroup.ID].Ports[port] = true
				}
			}
		} else {
			selectedSecurityGroups, err := SelectSecurityGroups(
				ctx,
				cfg,
				strings.Title(strings.ToLower(envOrIDOrName)),
				&filtersToApply,
				skipChoice,
				portsToUpdate,
			)
			if err != nil {
				return err
			}

			securityGroups = selectedSecurityGroups
		}
	}

	if len(securityGroups) == 0 {
		logger.Warn("No security group matched. Exiting")
		return nil
	}

	publicIP := utils.GetPublicIP()

	for _, sgAlter := range securityGroups {
		ports := make([]int32, 0)
		for port := range sgAlter.Ports {
			ports = append(ports, port)
		}

		logger.Info("Processing %v ports for %s (%s)", logger.Bold(ports), logger.Underline(sgAlter.SecurityGroup.Name), sgAlter.SecurityGroup.ID)

		sgRules := make([]SecurityGroupRule, 0)
		for port := range sgAlter.Ports {
			sgRule, _ := NewSecurityGroupRule(port, securityGroupUser)
			sgRules = append(sgRules, *sgRule)
		}

		rules := sgAlter.SecurityGroup.FilterIngressRules(&sgRules)
		if len(rules) > 0 {
			if err := sgAlter.SecurityGroup.Revoke(ctx, cfg, rules, publicIP, authorize); err != nil {
				logger.Error("Error on revoking rules for %s (%s). Error: %s", sgAlter.SecurityGroup.Name, sgAlter.SecurityGroup.ID, err.Error())
				continue
			}
		}

		sgAlter.SecurityGroup.Refresh(ctx, cfg)

		if authorize {
			sgAlter.SecurityGroup.Authorize(ctx, cfg, &sgRules, publicIP)
		}
	}

	return nil
}
