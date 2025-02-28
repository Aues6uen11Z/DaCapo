import enUS from './en-US';
import zhCN from './zh-CN';

export default {
  'en-US': enUS,
  'zh-CN': zhCN,
};


import { useIstStore } from '../stores/global-store';

export function useTranslation(instanceName: string) {
  const istStore = useIstStore();
  
  const getMenuName = (menuName: string) => {
    return istStore.getTranslatedText(instanceName, `${menuName}.name`) || menuName;
  };
  
  const getTaskName = (menuName: string, taskName: string) => {
    return istStore.getTranslatedText(instanceName, `${menuName}.tasks.${taskName}.name`) || taskName;
  };
  
  const getGroupName = (menuName: string, taskName: string, groupName: string) => {
    return istStore.getTranslatedText(instanceName, `${menuName}.tasks.${taskName}.groups.${groupName}.name`) || groupName;
  };
  
  const getGroupHelp = (menuName: string, taskName: string, groupName: string) => {
    return istStore.getTranslatedText(instanceName, `${menuName}.tasks.${taskName}.groups.${groupName}.help`) || '';
  };
  
  const getItemName = (menuName: string, taskName: string, groupName: string, itemName: string) => {
    return istStore.getTranslatedText(instanceName, `${menuName}.tasks.${taskName}.groups.${groupName}.items.${itemName}.name`) || itemName;
  };
  
  const getItemHelp = (menuName: string, taskName: string, groupName: string, itemName: string) => {
    return istStore.getTranslatedText(instanceName, `${menuName}.tasks.${taskName}.groups.${groupName}.items.${itemName}.help`) || '';
  };
  
  const getItemOption = (menuName: string, taskName: string, groupName: string, itemName: string, optionKey: string) => {
    return istStore.getTranslatedText(instanceName, `${menuName}.tasks.${taskName}.groups.${groupName}.items.${itemName}.options.${optionKey}`) || optionKey;
  };

  return {
    getMenuName,
    getTaskName,
    getGroupName,
    getGroupHelp,
    getItemName,
    getItemHelp,
    getItemOption,
  };
}