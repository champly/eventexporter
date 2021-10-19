package exporter

import (
	"github.com/champly/eventexporter/pkg/kube"
	"github.com/champly/eventexporter/pkg/sinks"
	"k8s.io/klog/v2"
)

// Route allows using rules to drop events or match events to specific receivers.
// It also allows using routes recursively for complex route building to fit
// most of the needs
type Route struct {
	Drop   []Rule
	Match  []Rule
	Routes []Route
}

func (r *Route) ProcessEvent(ev *kube.EnhancedEvent) {
	// First determine whether we will drop the event: If any of the drop is matched, we break the loop
	for _, v := range r.Drop {
		if v.MatchesEvent(ev) {
			klog.V(4).Infof("Drop event %s/%s", ev.Namespace, ev.Name)
			return
		}
	}

	// It has match rules, it should go to the matchers
	matchedAll := true
	for _, rule := range r.Match {
		if rule.MatchesEvent(ev) {
			if rule.Receiver != "" {
				klog.V(4).Infof("Send event %s/%s to %s receiver.", ev.Namespace, ev.Name, rule.Receiver)
				sinks.SendEvent(rule.Receiver, ev)
				// Send the event down the hole
			}
		} else {
			matchedAll = false
		}
	}

	// If all matches are satisfied, we can send them down to the rabbit hole
	if matchedAll {
		for _, subRoute := range r.Routes {
			klog.V(4).Infof("Send event %s/%s down to the rabbit hole.", ev.Namespace, ev.Name)
			subRoute.ProcessEvent(ev)
		}
	}
}
